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

	"github.com/alphagov/router/handlers"
	"github.com/alphagov/router/logger"
	"github.com/alphagov/router/triemux"
)

//go:embed sql/routes.sql
var loadRoutesQuery string

type PgxIface interface {
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
}

func addHandler(mux *triemux.Mux, route *CsRoute, backends map[string]http.Handler) error {
	if route.IncomingPath == nil || route.RouteType == nil {
		logWarn(fmt.Sprintf("router: found route %+v with nil fields, skipping!", route))
		return nil
	}

	prefix := (*route.RouteType == RouteTypePrefix)

	// the database contains paths with % encoded routes.
	// Unescape them here because the http.Request objects we match against contain the unescaped variants.
	incomingURL, err := url.Parse(*route.IncomingPath)
	if err != nil {
		logWarn(fmt.Sprintf("router: found route %+v with invalid incoming path '%s', skipping!", route, *route.IncomingPath))
		return nil //nolint:nilerr
	}

	switch route.handlerType() {
	case HandlerTypeBackend:
		backend := route.backend()
		if backend == nil {
			logWarn(fmt.Sprintf("router: found route %+v with nil backend_id, skipping!", *route.IncomingPath))
			return nil
		}
		handler, ok := backends[*backend]
		if !ok {
			logWarn(fmt.Sprintf("router: found route %+v with unknown backend "+
				"%s, skipping!", *route.IncomingPath, *route.BackendID))
			return nil
		}
		mux.Handle(incomingURL.Path, prefix, handler)
	case HandlerTypeRedirect:
		if route.RedirectTo == nil {
			logWarn(fmt.Sprintf("router: found route %+v with nil redirect_to, skipping!", *route.IncomingPath))
			return nil
		}
		handler := handlers.NewRedirectHandler(incomingURL.Path, *route.RedirectTo, shouldPreserveSegments(*route.RouteType, route.segmentsMode()))
		mux.Handle(incomingURL.Path, prefix, handler)
	case HandlerTypeGone:
		mux.Handle(incomingURL.Path, prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "410 Gone", http.StatusGone)
		}))
	default:
		logWarn(fmt.Sprintf("router: found route %+v with unknown handler type "+
			"%s, skipping!", route, route.handlerType()))
		return nil
	}
	return nil
}

func loadRoutesFromCS(pool PgxIface, mux *triemux.Mux, backends map[string]http.Handler) error {
	rows, err := pool.Query(context.Background(), loadRoutesQuery)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		route := &CsRoute{}
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

		err = addHandler(mux, route, backends)
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

	listener.Handle("route_changes", pgxlisten.HandlerFunc(
		func(ctx context.Context, notification *pgconn.Notification, conn *pgx.Conn) error {
			rt.CsReloadChan <- true
			return nil
		}),
	)

	err := listener.Listen(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (rt *Router) PeriodicCSRouteUpdates() {
	tick := time.Tick(time.Minute)
	for range tick {
		if time.Since(rt.csLastReloadTime) > time.Minute {
			rt.CsReloadChan <- true
		}
	}
}

func (rt *Router) waitForReload() {
	for range rt.CsReloadChan {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logWarn(r)
				}
			}()

			rt.reloadCsRoutes(rt.pool)
		}()
	}
}

func (rt *Router) reloadCsRoutes(pool PgxIface) {
	var success bool
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		labels := prometheus.Labels{"success": strconv.FormatBool(success), "source": "content-store"}
		routeReloadDurationMetric.With(labels).Observe(v)
	}))

	defer func() {
		success = true
		if r := recover(); r != nil {
			success = false
			logWarn("router: recovered from panic in reloadCsRoutes:", r)
			logInfo("router: original content store routes have not been modified")
			errorMessage := fmt.Sprintf("panic: %v", r)
			err := logger.RecoveredError{ErrorMessage: errorMessage}
			logger.NotifySentry(logger.ReportableError{Error: err})
		}
		timer.ObserveDuration()
	}()

	logInfo("router: reloading routes from content store")
	newmux := triemux.NewMux()

	err := loadRoutesFromCS(pool, newmux, rt.backends)
	if err != nil {
		logWarn(fmt.Sprintf("router: error reloading routes from content store: %v", err))
		return
	}

	routeCount := newmux.RouteCount()

	rt.lock.Lock()
	rt.csMux = newmux
	rt.lock.Unlock()

	logInfo(fmt.Sprintf("router: reloaded %d routes from content store", routeCount))
	routesCountMetric.WithLabelValues("content-store").Set(float64(routeCount))

}
