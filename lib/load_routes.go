package router

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jackc/pgx/v5"

	"github.com/alphagov/router/handlers"
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
		return nil
	}

	switch route.handlerType() {
	case "backend":
		backend := route.backend()
		if backend == nil {
			logWarn(fmt.Sprintf("router: found route %+v with nil backend_id, skipping!", *route.IncomingPath))
			return nil
		}
		handler, ok := backends[*backend]
		if !ok {
			logWarn(fmt.Sprintf("router: found route %+v which references unknown backend "+
				"%s, skipping!", route, *route.BackendID))
			return nil
		}
		mux.Handle(incomingURL.Path, prefix, handler)
	case "redirect":
		if route.RedirectTo == nil {
			logWarn(fmt.Sprintf("router: found route %+v with nil redirect_to, skipping!", route))
			return nil
		}
		handler := handlers.NewRedirectHandler(incomingURL.Path, *route.RedirectTo, shouldPreserveSegments(*route.RouteType, route.segmentsMode()))
		mux.Handle(incomingURL.Path, prefix, handler)
	case "gone":
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
