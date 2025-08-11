package closehandler

import (
	"net"
	"syscall"

	"github.com/caddyserver/caddy/v2"
	"github.com/mholt/caddy-l4/layer4"
)

func init() {
	caddy.RegisterModule(CloseHandler{})
}

type CloseHandler struct{}

// CaddyModule 返回模块信息
func (CloseHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "layer4.handlers.close",
		New: func() caddy.Module { return new(CloseHandler) },
	}
}

// Handle 实现 layer4.NextHandler 接口
// 关闭连接时设置 SO_LINGER=0，强制发送 TCP RST，立即关闭连接
func (h *CloseHandler) Handle(conn *layer4.Connection, next layer4.Handler) error {
	rawConn := conn.Conn

	tcpConn, ok := rawConn.(interface {
		SyscallConn() (syscall.RawConn, error)
	})
	if !ok {
		// 不是 TCP 连接则普通关闭
		return rawConn.Close()
	}

	sysConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}

	var serr error
	err = sysConn.Control(func(fd uintptr) {
		linger := syscall.Linger{
			Onoff:  1,
			Linger: 0, // 立刻发送RST
		}
		serr = syscall.SetsockoptLinger(int(fd), syscall.SOL_SOCKET, syscall.SO_LINGER, &linger)
	})
	if err != nil {
		return err
	}
	if serr != nil {
		return serr
	}

	return rawConn.Close()
}

var _ layer4.NextHandler = (*CloseHandler)(nil)
