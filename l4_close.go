package closehandler

import (
	"net"
	"syscall"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/mholt/caddy-l4/layer4"
)

func init() {
	caddy.RegisterModule(CloseHandler{})
}

type CloseHandler struct {
	// Timeout 指在进入下一 handler 前，等待数据的最大时间
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// MinRead 指在 Timeout 内，socket 中至少应可读的字节数
	MinRead int `json:"min_read,omitempty"`
}

// CaddyModule 返回模块信息
func (CloseHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "layer4.handlers.close",
		New: func() caddy.Module { return new(CloseHandler) },
	}
}

// Handle 实现 layer4.NextHandler 接口
// - 若配置了 Timeout / MinRead：
//   - 在指定时间内 peek 指定字节
//   - 不满足条件则 SO_LINGER=0 → TCP RST → close
// - 否则行为与原始版本完全一致
func (h *CloseHandler) Handle(conn *layer4.Connection, next layer4.Handler) error {
	rawConn := conn.Conn

	// ====== 超时 + 最小读取保护（仅在配置时启用） ======
	if h.Timeout > 0 || h.MinRead > 0 {
		deadline := time.Time{}
		if h.Timeout > 0 {
			deadline = time.Now().Add(time.Duration(h.Timeout))
			_ = rawConn.SetReadDeadline(deadline)
		}

		ok := h.peekEnough(rawConn)

		// 清除 deadline
		_ = rawConn.SetReadDeadline(time.Time{})

		if !ok {
			h.forceRST(rawConn)
			return nil
		}
	}

	// ====== 放行 ======
	if next != nil {
		return next.Handle(conn)
	}

	// ====== fallback：保持原有 close 行为 ======
	h.forceRST(rawConn)
	return nil
}

// peekEnough 使用 MSG_PEEK 检查 socket 中是否有足够数据
func (h *CloseHandler) peekEnough(rawConn net.Conn) bool {
	if h.MinRead <= 0 {
		return true
	}

	tcpConn, ok := rawConn.(interface {
		SyscallConn() (syscall.RawConn, error)
	})
	if !ok {
		return false
	}

	sysConn, err := tcpConn.SyscallConn()
	if err != nil {
		return false
	}

	var n int
	var serr error

	err = sysConn.Control(func(fd uintptr) {
		buf := make([]byte, h.MinRead)
		n, serr = syscall.Recvfrom(
			int(fd),
			buf,
			syscall.MSG_PEEK,
		)
	})
	if err != nil || serr != nil {
		return false
	}

	return n >= h.MinRead
}

// forceRST 通过 SO_LINGER=0 强制发送 TCP RST 并关闭连接
func (h *CloseHandler) forceRST(rawConn net.Conn) {
	tcpConn, ok := rawConn.(interface {
		SyscallConn() (syscall.RawConn, error)
	})
	if !ok {
		_ = rawConn.Close()
		return
	}

	sysConn, err := tcpConn.SyscallConn()
	if err == nil {
		_ = sysConn.Control(func(fd uintptr) {
			linger := syscall.Linger{
				Onoff:  1,
				Linger: 0, // 立刻发送 RST
			}
			_ = syscall.SetsockoptLinger(
				int(fd),
				syscall.SOL_SOCKET,
				syscall.SO_LINGER,
				&linger,
			)
		})
	}

	_ = rawConn.Close()
}

var _ layer4.NextHandler = (*CloseHandler)(nil)
