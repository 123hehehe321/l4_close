package closehandler

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/mholt/caddy-l4/layer4"
)

func init() {
	caddy.RegisterModule(CloseHandler{})
}

type CloseHandler struct{}

// CaddyModule returns the Caddy module information.
func (CloseHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "layer4.handlers.close",
		New: func() caddy.Module { return new(CloseHandler) },
	}
}

// Serve implements layer4.NextHandler.
func (h *CloseHandler) Serve(conn *layer4.Connection) error {
	// 直接关闭底层连接
	return conn.Close()
}

// 确保实现了 layer4.NextHandler 接口
var _ layer4.NextHandler = (*CloseHandler)(nil)
