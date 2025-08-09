package closehandler

import (
	"github.com/mholt/caddy-l4/layer4"
	"github.com/caddyserver/caddy/v2"
)

func init() {
	caddy.RegisterModule(CloseHandler{})
}

type CloseHandler struct{}

func (CloseHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "layer4.handlers.close",
		New: func() caddy.Module { return new(CloseHandler) },
	}
}

func (h *CloseHandler) Serve(conn layer4.Conn) error {
	conn.Close()
	return nil
}
