package closehandler

import (
	"github.com/mholt/caddy-l4/layer4"
	"github.com/caddyserver/caddy/v2"
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

// Serve implements layer4.Handler.
func (h *CloseHandler) Serve(ctx *layer4.HandlerContext) error {
	// 直接关闭底层连接
	return ctx.Conn.Close()
}

// 确保实现了 layer4.Handler 接口
var _ layer4.Handler = (*CloseHandler)(nil)
