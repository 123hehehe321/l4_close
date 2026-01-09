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
	// Timeout 指在进入下一 handler 之前，允许读取数据的最大时间
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// MinRead 指在 Timeout 时间内，至少需要读取的字节数
	// 小于该值将被视为异常连接并立即 RST
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
// 在满足超时 / 读取条件前：
//   - 不满足条件 → 设置 SO_LINGER=0，发送 TCP RST，立即关闭连接
//   - 满足条件 → 放行到下一个 handler（如 proxy）
//
// 若未配置 Timeout / MinRead，则行为与原始版本完全一致
func (h *CloseHandler) Handle(conn *layer4.Connection, next layer4.Handler) error {
	rawConn := conn.Conn

	// ====== 1. 超时 + 最小读取保护（仅在配置时启用） ======
	if h.Timeout > 0 || h.MinRead > 0 {
		// 设置读超时
		if h.Timeout > 0 {
			_ = rawConn.SetReadDeadline(time.Now().Add(time.Duration(h.Timeout)))
		}

		// 若要求最小读取
		if h.MinRead > 0 {
			buf := make([]byte, h.MinRead)
			n, err := rawConn.Read(buf)
			if err != nil || n < h.MinRead {
				h.forceRST(rawConn)
				return nil
			}

			// 将读取的数据塞回 buffer，确保不影响后续 handler（非常关键）
			conn.Buffer = append(buf[:n], conn.Buffer...)
		}

		// 清除读超时
		_ = rawConn.SetReadDeadline(time.Time{})
	}

	// ====== 2. 放行到下一个 handler（如 proxy） ======
	if next != nil {
		return next.Handle(conn)
	}

	// ====== 3. fallback：保持原有 close 行为 ======
	h.forceRST(rawConn)
	return nil
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
			_ = syscall.SetsockoptLinger(int(fd), syscall.SOL_SOCKET, syscall.SO_LINGER, &linger)
		})
	}

	_ = rawConn.Close()
}

var _ layer4.NextHandler = (*CloseHandler)(nil)
