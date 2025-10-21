package router

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgxlisten"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/alphagov/router/handlers"
	"github.com/alphagov/router/triemux"
)

//go:embed sql/routes.sql
var loadRoutesQuery string

type PgxIface interface {
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
}

func addHandler(mux *triemux.Mux, route *Route, backends map[string]http.Handler, logger zerolog.Logger) error {
	if route.IncomingPath == nil || route.RouteType == nil {
		logger.Warn().Interface("route", route).Msg("ignoring route with nil fields")
		return nil
	}

	prefix := (*route.RouteType == RouteTypePrefix)

	incomingURL, err := url.Parse(*route.IncomingPath)
	if err != nil {
		logger.Warn().Interface("route", route).Str("incoming_path", *route.IncomingPath).Msg("ignoring route with invalid incoming path")
		return nil //nolint:nilerr
	}

	switch route.handlerType() {
	case HandlerTypeBackend:
		backend := route.backend()
		if backend == nil {
			logger.Warn().Str("incoming_path", *route.IncomingPath).Msg("ignoring route with nil backend_id")
			return nil
		}
		handler, ok := backends[*backend]
		if !ok {
			logger.Warn().Str("incoming_path", *route.IncomingPath).Str("backend_id", *route.BackendID).Msg("ignoring route with unknown backend")
			return nil
		}
		mux.Handle(incomingURL.Path, prefix, handler)
	case HandlerTypeRedirect:
		if route.RedirectTo == nil {
			logger.Warn().Str("incoming_path", *route.IncomingPath).Msg("ignoring route with nil redirect_to")
			return nil
		}
		handler := handlers.NewRedirectHandler(incomingURL.Path, *route.RedirectTo, shouldPreserveSegments(*route.RouteType, route.segmentsMode()), logger)
		mux.Handle(incomingURL.Path, prefix, handler)
	case HandlerTypeGone:
		mux.Handle(incomingURL.Path, prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "410 Gone", http.StatusGone)
		}))
	default:
		logger.Warn().Interface("route", route).Str("handler_type", route.handlerType()).Msg("ignoring route with unknown handler type")
		return nil
	}
	return nil
}

func loadRoutes(pool PgxIface, mux *triemux.Mux, backends map[string]http.Handler, logger zerolog.Logger) error {
	rows, err := pool.Query(context.Background(), loadRoutesQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		route := &Route{}
		scans := []any{
			&route.BackendID,
			&route.IncomingPath,
			&route.RouteType,
			&route.RedirectTo,
			&route.SegmentsMode,
			&route.SchemaName,
			&route.Details,
		}

		err := rows.Scan(scans...)
		if err != nil {
			return err
		}

		err = addHandler(mux, route, backends, logger)
		if err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func (rt *Router) listenForContentStoreUpdates(ctx context.Context) error {
	listener := &pgxlisten.Listener{
		Connect: func(ctx context.Context) (*pgx.Conn, error) {
			c, err := rt.pool.Acquire(ctx)
			if err != nil {
				return nil, err
			}
			return c.Conn(), nil
		},
	}

	listener.Handle(
		"route_changes",
		pgxlisten.HandlerFunc(
			func(ctx context.Context, notification *pgconn.Notification, conn *pgx.Conn) error {
				// This is a non-blocking send, if there is already a notification to reload we don't need to send another one
				select {
				case rt.ReloadChan <- true:
				default:
				}
				return nil
			},
		),
	)

	err := listener.Listen(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (rt *Router) PeriodicRouteUpdates() {
	// Skip periodic updates if ReloadChan is nil (e.g., when using flat file)
	if rt.ReloadChan == nil {
		return
	}

	tick := time.Tick(5 * time.Second)
	for range tick {
		if time.Since(rt.lastAttemptReloadTime) > rt.opts.RouteReloadInterval {
			// This is a non-blocking send, if there is already a notification to reload we don't need to send another one
			select {
			case rt.ReloadChan <- true:
			default:
			}
		}
	}
}

func (rt *Router) waitForReload() {
	for range rt.ReloadChan {
		rt.reloadRoutes(rt.pool)
	}
}

func (rt *Router) reloadRoutes(pool PgxIface) {
	var success bool
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		labels := prometheus.Labels{"success": strconv.FormatBool(success), "source": "content-store"}
		routeReloadDurationMetric.With(labels).Observe(v)
	}))

	defer func() {
		success = true
		if r := recover(); r != nil {
			success = false
			rt.Logger.Err(fmt.Errorf("%v", r)).Msgf("recovered from panic in reloadRoutes")
			rt.Logger.Info().Msg("reload failed and existing routes have not been modified")
		}
		timer.ObserveDuration()
	}()

	rt.lastAttemptReloadTime = time.Now()

	rt.Logger.Info().Msg("reloading routes from content store")
	newmux := triemux.NewMux(rt.Logger)

	err := loadRoutes(pool, newmux, rt.backends, rt.Logger)
	if err != nil {
		rt.Logger.Warn().Err(err).Msg("error reloading routes")
		return
	}

	routeCount := newmux.RouteCount()

	rt.lock.Lock()
	rt.mux = newmux
	rt.lock.Unlock()

	rt.Logger.Info().Int("route_count", routeCount).Msg("reloaded routes")
	routesCountMetric.WithLabelValues("content-store").Set(float64(routeCount))
}
