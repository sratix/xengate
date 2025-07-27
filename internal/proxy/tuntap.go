package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"xengate/internal/tunnel"

	log "github.com/sirupsen/logrus"
	"github.com/songgao/water"
)

type TunTapProxy struct {
	manager    *tunnel.Manager
	ifce       *water.Interface
	netManager *NetworkManager
	wg         sync.WaitGroup
	mu         sync.RWMutex
	closed     bool
	ip         string
	port       int16
	name       string
	active     int64
	totalBytes int64
}

type NetworkManager struct {
	iface         string
	ip            string
	origSysctl    map[string]string
	iptablesRules []string
	routes        []string
	mu            sync.Mutex
}

func init() {
	// تنظیم سطح لاگ
	log.SetLevel(log.DebugLevel)
}

func NewTunTapProxy(name, ip string, port int16, manager *tunnel.Manager) (*TunTapProxy, error) {
	// چک کردن دسترسی root
	if os.Geteuid() != 0 {
		return nil, fmt.Errorf("این برنامه باید با دسترسی root اجرا شود")
	}

	netManager := NewNetworkManager(name, ip)
	return &TunTapProxy{
		manager:    manager,
		ip:         ip,
		port:       port,
		name:       name,
		netManager: netManager,
	}, nil
}

func NewNetworkManager(iface, ip string) *NetworkManager {
	return &NetworkManager{
		iface:      iface,
		ip:         ip,
		origSysctl: make(map[string]string),
		iptablesRules: []string{
			"INPUT -i %s -j ACCEPT",
			"OUTPUT -o %s -j ACCEPT",
			"nat -t nat -A POSTROUTING -o eth0 -j MASQUERADE",
			"FORWARD -i %s -o eth0 -j ACCEPT",
			"FORWARD -i eth0 -o %s -m state --state RELATED,ESTABLISHED -j ACCEPT",
		},
		routes: []string{
			"0.0.0.0/0 via %s dev %s",
		},
	}
}

func (t *TunTapProxy) Start(ctx context.Context) error {
	log.Info("شروع پروکسی TUN/TAP...")

	// ایجاد رابط TUN
	config := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: t.name,
		},
	}

	ifce, err := water.New(config)
	if err != nil {
		return fmt.Errorf("خطا در ایجاد رابط TUN: %w", err)
	}

	t.ifce = ifce
	log.Infof("رابط TUN %s ایجاد شد", ifce.Name())

	// تنظیم پیکربندی شبکه
	if err := t.netManager.Setup(); err != nil {
		t.ifce.Close()
		return fmt.Errorf("خطا در تنظیم شبکه: %w", err)
	}

	// تنظیم آدرس IP
	if err := t.configureInterface(); err != nil {
		t.netManager.Cleanup()
		t.ifce.Close()
		return fmt.Errorf("خطا در تنظیم رابط: %w", err)
	}

	// شروع پردازش بسته‌ها
	t.wg.Add(1)
	go t.handlePackets(ctx)

	// مدیریت خاموش شدن
	go func() {
		<-ctx.Done()
		t.Stop()
	}()

	log.Info("پروکسی TUN/TAP با موفقیت شروع شد")
	return nil
}

func (t *TunTapProxy) configureInterface() error {
	cmd := exec.Command("ip", "addr", "add", t.ip+"/24", "dev", t.name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("خطا در تنظیم آدرس IP: %w", err)
	}

	cmd = exec.Command("ip", "link", "set", "dev", t.name, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("خطا در فعال کردن رابط: %w", err)
	}

	return nil
}

func (nm *NetworkManager) Setup() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	log.Info("تنظیم پیکربندی شبکه...")

	// ذخیره مقادیر اصلی sysctl
	sysctls := []string{
		"net.ipv4.ip_forward",
		"net.core.rmem_max",
		"net.core.wmem_max",
	}

	for _, s := range sysctls {
		val, err := nm.getSysctl(s)
		if err != nil {
			return fmt.Errorf("خطا در خواندن %s: %w", s, err)
		}
		nm.origSysctl[s] = val
		log.Debugf("مقدار اصلی %s = %s ذخیره شد", s, val)
	}

	// فعال کردن IP forwarding و تنظیم پارامترهای شبکه
	if err := nm.setSysctl("net.ipv4.ip_forward", "1"); err != nil {
		return err
	}
	if err := nm.setSysctl("net.core.rmem_max", "26214400"); err != nil {
		return err
	}
	if err := nm.setSysctl("net.core.wmem_max", "26214400"); err != nil {
		return err
	}

	// تنظیم قوانین iptables
	for _, rule := range nm.iptablesRules {
		rule = fmt.Sprintf(rule, nm.iface)
		if strings.Contains(rule, "nat") {
			args := strings.Split(rule, " ")
			cmd := exec.Command("iptables", args...)
			if output, err := cmd.CombinedOutput(); err != nil {
				log.WithError(err).Errorf("خطا در اجرای دستور iptables: %s", string(output))
				return fmt.Errorf("خطا در تنظیم قانون iptables: %w", err)
			}
		} else {
			args := append([]string{"-A"}, strings.Split(rule, " ")...)
			cmd := exec.Command("iptables", args...)
			if output, err := cmd.CombinedOutput(); err != nil {
				log.WithError(err).Errorf("خطا در اجرای دستور iptables: %s", string(output))
				return fmt.Errorf("خطا در تنظیم قانون iptables: %w", err)
			}
		}
		log.Debugf("قانون iptables اضافه شد: %s", rule)
	}

	// تنظیم مسیریابی
	for _, route := range nm.routes {
		route = fmt.Sprintf(route, nm.ip, nm.iface)
		args := append([]string{"route", "add"}, strings.Split(route, " ")...)
		cmd := exec.Command("ip", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.WithError(err).Errorf("خطا در اجرای دستور ip route: %s", string(output))
			return fmt.Errorf("خطا در تنظیم مسیریابی: %w", err)
		}
		log.Debugf("مسیر اضافه شد: %s", route)
	}

	log.Info("پیکربندی شبکه با موفقیت انجام شد")
	return nil
}

func (nm *NetworkManager) Cleanup() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	log.Info("پاکسازی پیکربندی شبکه...")

	// بازگرداندن مقادیر اصلی sysctl
	for k, v := range nm.origSysctl {
		if err := nm.setSysctl(k, v); err != nil {
			log.WithError(err).Warnf("خطا در بازگرداندن %s", k)
		} else {
			log.Debugf("مقدار %s به %s بازگردانده شد", k, v)
		}
	}

	// حذف قوانین iptables
	for _, rule := range nm.iptablesRules {
		rule = fmt.Sprintf(rule, nm.iface)
		if strings.Contains(rule, "nat") {
			args := strings.Split(strings.Replace(rule, "-A", "-D", 1), " ")
			cmd := exec.Command("iptables", args...)
			if err := cmd.Run(); err != nil {
				log.WithError(err).Warnf("خطا در حذف قانون iptables: %s", rule)
			}
		} else {
			args := append([]string{"-D"}, strings.Split(rule, " ")...)
			cmd := exec.Command("iptables", args...)
			if err := cmd.Run(); err != nil {
				log.WithError(err).Warnf("خطا در حذف قانون iptables: %s", rule)
			}
		}
		log.Debugf("قانون iptables حذف شد: %s", rule)
	}

	// حذف مسیرها
	for _, route := range nm.routes {
		route = fmt.Sprintf(route, nm.ip, nm.iface)
		args := append([]string{"route", "del"}, strings.Split(route, " ")...)
		cmd := exec.Command("ip", args...)
		if err := cmd.Run(); err != nil {
			log.WithError(err).Warnf("خطا در حذف مسیر: %s", route)
		} else {
			log.Debugf("مسیر حذف شد: %s", route)
		}
	}

	log.Info("پاکسازی شبکه تکمیل شد")
	return nil
}

func (nm *NetworkManager) setSysctl(key, value string) error {
	log.Debugf("تنظیم sysctl %s = %s", key, value)
	cmd := exec.Command("sysctl", "-w", fmt.Sprintf("%s=%s", key, value))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("خطا در تنظیم sysctl: %w: %s", err, string(output))
	}
	return nil
}

func (nm *NetworkManager) getSysctl(key string) (string, error) {
	cmd := exec.Command("sysctl", "-n", key)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("خطا در خواندن sysctl: %w: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func (t *TunTapProxy) handlePackets(ctx context.Context) {
	defer t.wg.Done()

	packet := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := t.ifce.Read(packet)
			if err != nil {
				if err != io.EOF && !strings.Contains(err.Error(), "closed") {
					log.Errorf("خطا در خواندن از TUN: %v", err)
				}
				return
			}

			atomic.AddInt64(&t.totalBytes, int64(n))

			if n < 20 {
				continue // خیلی کوتاه برای هدر IP
			}

			version := packet[0] >> 4
			if version != 4 {
				continue // فقط IPv4 پشتیبانی می‌شود
			}

			protocol := packet[9]
			srcIP := net.IP(packet[12:16])
			dstIP := net.IP(packet[16:20])

			switch protocol {
			case 6: // TCP
				t.handleTCP(packet[:n], srcIP, dstIP)
			case 17: // UDP
				t.handleUDP(packet[:n], srcIP, dstIP)
			}
		}
	}
}

func (t *TunTapProxy) handleTCP(packet []byte, srcIP, dstIP net.IP) {
	srcPort := uint16(packet[20])<<8 | uint16(packet[21])
	dstPort := uint16(packet[22])<<8 | uint16(packet[23])

	target := fmt.Sprintf("%s:%d", dstIP.String(), dstPort)
	log.Debugf("TCP: %s:%d -> %s", srcIP, srcPort, target)

	clientConn, serverConn := net.Pipe()

	atomic.AddInt64(&t.active, 1)
	defer atomic.AddInt64(&t.active, -1)

	go func() {
		defer clientConn.Close()
		defer serverConn.Close()

		if err := t.manager.Forward(serverConn, target); err != nil {
			if !isNormalError(err) {
				log.Debugf("خطا در انتقال TCP: %v", err)
			}
		}
	}()

	go func() {
		if _, err := clientConn.Write(packet[20:]); err != nil && !isNormalError(err) {
			log.Debugf("خطا در نوشتن در تونل: %v", err)
			return
		}
	}()

	response := make([]byte, 1500)
	for {
		n, err := clientConn.Read(response[20:])
		if err != nil {
			if !isNormalError(err) {
				log.Debugf("خطا در خواندن از تونل: %v", err)
			}
			return
		}

		t.buildIPHeader(response[:20], dstIP, srcIP, 6, uint16(n))

		if _, err := t.ifce.Write(response[:20+n]); err != nil && !isNormalError(err) {
			log.Debugf("خطا در نوشتن در TUN: %v", err)
			return
		}
	}
}

func (t *TunTapProxy) handleUDP(packet []byte, srcIP, dstIP net.IP) {
	srcPort := uint16(packet[20])<<8 | uint16(packet[21])
	dstPort := uint16(packet[22])<<8 | uint16(packet[23])

	target := fmt.Sprintf("%s:%d", dstIP.String(), dstPort)
	log.Debugf("UDP: %s:%d -> %s", srcIP, srcPort, target)

	clientConn, serverConn := net.Pipe()

	atomic.AddInt64(&t.active, 1)
	defer atomic.AddInt64(&t.active, -1)

	go func() {
		defer clientConn.Close()
		defer serverConn.Close()

		if err := t.manager.Forward(serverConn, target); err != nil {
			if !isNormalError(err) {
				log.Debugf("خطا در انتقال UDP: %v", err)
			}
		}
	}()

	go func() {
		if _, err := clientConn.Write(packet[20:]); err != nil && !isNormalError(err) {
			log.Debugf("خطا در نوشتن در تونل: %v", err)
			return
		}
	}()

	response := make([]byte, 1500)
	for {
		n, err := clientConn.Read(response[20:])
		if err != nil {
			if !isNormalError(err) {
				log.Debugf("خطا در خواندن از تونل: %v", err)
			}
			return
		}

		t.buildIPHeader(response[:20], dstIP, srcIP, 17, uint16(n))

		if _, err := t.ifce.Write(response[:20+n]); err != nil && !isNormalError(err) {
			log.Debugf("خطا در نوشتن در TUN: %v", err)
			return
		}
	}
}

func (t *TunTapProxy) buildIPHeader(header []byte, srcIP, dstIP net.IP, protocol uint8, length uint16) {
	header[0] = 0x45 // نسخه 4، طول هدر 5 کلمه
	header[1] = 0x00 // DSCP/ECN
	header[2] = byte(length >> 8)
	header[3] = byte(length)
	header[4] = 0x00 // شناسه
	header[5] = 0x00
	header[6] = 0x00 // پرچم‌ها/آفست قطعه
	header[7] = 0x00
	header[8] = 64 // TTL
	header[9] = protocol
	header[10] = 0x00 // چکسام (باید محاسبه شود)
	header[11] = 0x00
	copy(header[12:16], srcIP.To4())
	copy(header[16:20], dstIP.To4())

	// محاسبه چکسام
	var sum uint32
	for i := 0; i < 20; i += 2 {
		sum += uint32(header[i])<<8 | uint32(header[i+1])
	}
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16
	header[10] = byte(^sum >> 8)
	header[11] = byte(^sum)
}

func (t *TunTapProxy) Stop() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()

	log.Info("توقف پروکسی TUN/TAP...")

	if err := t.netManager.Cleanup(); err != nil {
		log.WithError(err).Warn("خطا در پاکسازی پیکربندی شبکه")
	}

	if t.ifce != nil {
		if err := t.ifce.Close(); err != nil {
			log.WithError(err).Warn("خطا در بستن رابط TUN")
		}
	}

	t.wg.Wait()
	log.Info("پروکسی TUN/TAP متوقف شد")
	return nil
}

func (t *TunTapProxy) GetStats() TunTapStats {
	return TunTapStats{
		Active:     atomic.LoadInt64(&t.active),
		TotalBytes: atomic.LoadInt64(&t.totalBytes),
	}
}

type TunTapStats struct {
	Active     int64
	TotalBytes int64
}

func isNormalError(err error) bool {
	if err == nil {
		return true
	}
	return err == io.EOF ||
		strings.Contains(err.Error(), "closed") ||
		strings.Contains(err.Error(), "reset") ||
		strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "use of closed network connection")
}

// func (nm *NetworkManager) Setup() error {
//     nm.mu.Lock()
//     defer nm.mu.Unlock()

//     log.Info("تنظیم پیکربندی شبکه...")

//     // ذخیره مقادیر اصلی sysctl
//     sysctls := []string{
//         "net.ipv4.ip_forward",
//         "net.core.rmem_max",
//         "net.core.wmem_max",
//     }

//     for _, s := range sysctls {
//         val, err := nm.getSysctl(s)
//         if err != nil {
//             return fmt.Errorf("خطا در خواندن %s: %w", s, err)
//         }
//         nm.origSysctl[s] = val
//         log.Debugf("مقدار اصلی %s = %s ذخیره شد", s, val)
//     }

//     // فعال کردن IP forwarding و تنظیم پارامترهای شبکه
//     if err := nm.setSysctl("net.ipv4.ip_forward", "1"); err != nil {
//         return err
//     }
//     if err := nm.setSysctl("net.core.rmem_max", "26214400"); err != nil {
//         return err
//     }
//     if err := nm.setSysctl("net.core.wmem_max", "26214400"); err != nil {
//         return err
//     }

//     // تنظیم قوانین iptables
//     for _, rule := range nm.iptablesRules {
//         var cmd *exec.Cmd

//         if strings.Contains(rule, "-t nat") {
//             // برای قوانین NAT
//             args := strings.Split(rule, " ")
//             cmd = exec.Command("iptables", args...)
//         } else {
//             // برای قوانین عادی
//             rule = fmt.Sprintf(rule, nm.iface)
//             args := append([]string{"-A"}, strings.Split(rule, " ")...)
//             cmd = exec.Command("iptables", args...)
//         }

//         if output, err := cmd.CombinedOutput(); err != nil {
//             log.WithError(err).Errorf("خطا در اجرای دستور iptables: %s", string(output))
//             return fmt.Errorf("خطا در تنظیم قانون iptables: %w", err)
//         }
//         log.Debugf("قانون iptables اضافه شد: %s", rule)
//     }

//     // تنظیم مسیریابی
//     for _, route := range nm.routes {
//         route = fmt.Sprintf(route, nm.ip, nm.iface)
//         args := append([]string{"route", "add"}, strings.Split(route, " ")...)
//         cmd := exec.Command("ip", args...)
//         if output, err := cmd.CombinedOutput(); err != nil {
//             log.WithError(err).Errorf("خطا در اجرای دستور ip route: %s", string(output))
//             return fmt.Errorf("خطا در تنظیم مسیریابی: %w", err)
//         }
//         log.Debugf("مسیر اضافه شد: %s", route)
//     }

//     log.Info("پیکربندی شبکه با موفقیت انجام شد")
//     return nil
// }

// func (nm *NetworkManager) Cleanup() error {
//     nm.mu.Lock()
//     defer nm.mu.Unlock()

//     log.Info("پاکسازی پیکربندی شبکه...")

//     // بازگرداندن مقادیر اصلی sysctl
//     for k, v := range nm.origSysctl {
//         if err := nm.setSysctl(k, v); err != nil {
//             log.WithError(err).Warnf("خطا در بازگرداندن %s", k)
//         } else {
//             log.Debugf("مقدار %s به %s بازگردانده شد", k, v)
//         }
//     }

//     // حذف قوانین iptables
//     for _, rule := range nm.iptablesRules {
//         var cmd *exec.Cmd

//         if strings.Contains(rule, "-t nat") {
//             // برای قوانین NAT
//             args := strings.Split(strings.Replace(rule, "-A", "-D", 1), " ")
//             cmd = exec.Command("iptables", args...)
//         } else {
//             // برای قوانین عادی
//             rule = fmt.Sprintf(rule, nm.iface)
//             args := append([]string{"-D"}, strings.Split(rule, " ")...)
//             cmd = exec.Command("iptables", args...)
//         }

//         if err := cmd.Run(); err != nil {
//             log.WithError(err).Warnf("خطا در حذف قانون iptables: %s", rule)
//         } else {
//             log.Debugf("قانون iptables حذف شد: %s", rule)
//         }
//     }

//     // حذف مسیرها
//     for _, route := range nm.routes {
//         route = fmt.Sprintf(route, nm.ip, nm.iface)
//         args := append([]string{"route", "del"}, strings.Split(route, " ")...)
//         cmd := exec.Command("ip", args...)
//         if err := cmd.Run(); err != nil {
//             log.WithError(err).Warnf("خطا در حذف مسیر: %s", route)
//         } else {
//             log.Debugf("مسیر حذف شد: %s", route)
//         }
//     }

//     log.Info("پاکسازی شبکه تکمیل شد")
//     return nil
// }

// func (nm *NetworkManager) setSysctl(key, value string) error {
//     log.Debugf("تنظیم sysctl %s = %s", key, value)
//     cmd := exec.Command("sysctl", "-w", fmt.Sprintf("%s=%s", key, value))
//     if output, err := cmd.CombinedOutput(); err != nil {
//         return fmt.Errorf("خطا در تنظیم sysctl: %w: %s", err, string(output))
//     }
//     return nil
// }

// func (nm *NetworkManager) getSysctl(key string) (string, error) {
//     cmd := exec.Command("sysctl", "-n", key)
//     output, err := cmd.CombinedOutput()
//     if err != nil {
//         return "", fmt.Errorf("خطا در خواندن sysctl: %w: %s", err, string(output))
//     }
//     return strings.TrimSpace(string(output)), nil
// }

// func (t *TunTapProxy) configureInterface() error {
//     // تنظیم آدرس IP
//     cmd := exec.Command("ip", "addr", "add", t.ip+"/24", "dev", t.name)
//     if output, err := cmd.CombinedOutput(); err != nil {
//         return fmt.Errorf("خطا در تنظیم آدرس IP: %w: %s", err, string(output))
//     }

//     // فعال کردن رابط
//     cmd = exec.Command("ip", "link", "set", "dev", t.name, "up")
//     if output, err := cmd.CombinedOutput(); err != nil {
//         return fmt.Errorf("خطا در فعال کردن رابط: %w: %s", err, string(output))
//     }

//     // تنظیم MTU (اختیاری)
//     cmd = exec.Command("ip", "link", "set", "dev", t.name, "mtu", "1500")
//     if output, err := cmd.CombinedOutput(); err != nil {
//         log.Warnf("خطا در تنظیم MTU: %v: %s", err, string(output))
//     }

//     log.Infof("رابط %s با IP %s پیکربندی شد", t.name, t.ip)
//     return nil
// }

// type NetworkManager struct {
//     iface         string
//     ip            string
//     origSysctl    map[string]string
//     iptablesRules []string
//     routes        []string
//     mu            sync.Mutex
// }

// func NewNetworkManager(iface, ip string) *NetworkManager {
//     return &NetworkManager{
//         iface: iface,
//         ip:    ip,
//         origSysctl: make(map[string]string),
//         iptablesRules: []string{
//             // قوانین عادی
//             "INPUT -i %s -j ACCEPT",
//             "OUTPUT -o %s -j ACCEPT",
//             "FORWARD -i %s -o eth0 -j ACCEPT",
//             "FORWARD -i eth0 -o %s -m state --state RELATED,ESTABLISHED -j ACCEPT",
//             // قانون NAT به صورت جداگانه
//             "-t nat -A POSTROUTING -o eth0 -j MASQUERADE",
//         },
//         routes: []string{
//             "0.0.0.0/0 via %s dev %s",
//         },
//     }
// }

// type TunTapProxy struct {
//     manager    *tunnel.Manager
//     ifce       *water.Interface
//     netManager *NetworkManager
//     wg         sync.WaitGroup
//     mu         sync.RWMutex
//     closed     bool
//     ip         string
//     port       int16
//     name       string
//     active     int64
//     totalBytes int64
// }

// func NewTunTapProxy(name, ip string, port int16, manager *tunnel.Manager) (*TunTapProxy, error) {
//     // چک کردن دسترسی root
//     if os.Geteuid() != 0 {
//         return nil, fmt.Errorf("این برنامه باید با دسترسی root اجرا شود")
//     }

//     netManager := NewNetworkManager(name, ip)

//     return &TunTapProxy{
//         manager:    manager,
//         ip:         ip,
//         port:       port,
//         name:       name,
//         netManager: netManager,
//     }, nil
// }

// func (t *TunTapProxy) Start(ctx context.Context) error {
//     log.Info("شروع پروکسی TUN/TAP...")

//     // ایجاد رابط TUN
//     config := water.Config{
//         DeviceType: water.TUN,
//         PlatformSpecificParams: water.PlatformSpecificParams{
//             Name: t.name,
//         },
//     }

//     ifce, err := water.New(config)
//     if err != nil {
//         return fmt.Errorf("خطا در ایجاد رابط TUN: %w", err)
//     }

//     t.ifce = ifce
//     log.Infof("رابط TUN %s ایجاد شد", ifce.Name())

//     // تنظیم پیکربندی شبکه
//     if err := t.netManager.Setup(); err != nil {
//         t.ifce.Close()
//         return fmt.Errorf("خطا در تنظیم شبکه: %w", err)
//     }

//     // تنظیم آدرس IP
//     if err := t.configureInterface(); err != nil {
//         t.netManager.Cleanup()
//         t.ifce.Close()
//         return fmt.Errorf("خطا در تنظیم رابط: %w", err)
//     }

//     // شروع پردازش بسته‌ها
//     t.wg.Add(1)
//     go t.handlePackets(ctx)

//     // مدیریت خاموش شدن
//     go func() {
//         <-ctx.Done()
//         t.Stop()
//     }()

//     log.Info("پروکسی TUN/TAP با موفقیت شروع شد")
//     return nil
// }

// func (t *TunTapProxy) Stop() error {
//     t.mu.Lock()
//     if t.closed {
//         t.mu.Unlock()
//         return nil
//     }
//     t.closed = true
//     t.mu.Unlock()

//     log.Info("توقف پروکسی TUN/TAP...")

//     if err := t.netManager.Cleanup(); err != nil {
//         log.WithError(err).Warn("خطا در پاکسازی پیکربندی شبکه")
//     }

//     if t.ifce != nil {
//         if err := t.ifce.Close(); err != nil {
//             log.WithError(err).Warn("خطا در بستن رابط TUN")
//         }
//     }

//     t.wg.Wait()
//     log.Info("پروکسی TUN/TAP متوقف شد")
//     return nil
// }