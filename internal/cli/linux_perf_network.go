package cli

import (
	"strconv"
	"strings"
)

type netDevSample struct {
	rxBytes, txBytes, rxErr, txErr, rxDrop uint64
}

func parseNetDev() (map[string]netDevSample, error) {
	lines, err := readProcLines(procPath("net", "dev"))
	if err != nil {
		return nil, err
	}
	out := map[string]netDevSample{}
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "Inter-") || strings.HasPrefix(ln, "face") {
			continue
		}
		if !strings.Contains(ln, ":") {
			continue
		}
		parts := strings.Fields(strings.Replace(ln, ":", " ", 1))
		if len(parts) < 17 {
			continue
		}
		name := strings.TrimSuffix(parts[0], ":")
		if name == "lo" {
			continue
		}
		rx, _ := strconv.ParseUint(parts[1], 10, 64)
		rxErr, _ := strconv.ParseUint(parts[2], 10, 64)
		rxDrop, _ := strconv.ParseUint(parts[3], 10, 64)
		tx, _ := strconv.ParseUint(parts[9], 10, 64)
		txErr, _ := strconv.ParseUint(parts[10], 10, 64)
		out[name] = netDevSample{rxBytes: rx, txBytes: tx, rxErr: rxErr, txErr: txErr, rxDrop: rxDrop}
	}
	return out, nil
}

func networkDelta(a, b map[string]netDevSample, secs float64) linuxPerfNetwork {
	var out linuxPerfNetwork
	if secs <= 0 {
		secs = 1
	}
	for name, sb := range b {
		sa, ok := a[name]
		if !ok {
			sa = sb
		}
		rxBps := float64(sb.rxBytes-sa.rxBytes) / secs
		txBps := float64(sb.txBytes-sa.txBytes) / secs
		if rxBps < 0 {
			rxBps = 0
		}
		if txBps < 0 {
			txBps = 0
		}
		rxErr := sb.rxErr - sa.rxErr
		txErr := sb.txErr - sa.txErr
		out.TotalRxBps += rxBps
		out.TotalTxBps += txBps
		out.TotalRxErr += rxErr
		out.TotalTxErr += txErr
		out.TotalRxDrop += sb.rxDrop - sa.rxDrop
		if rxBps > 1024 || txBps > 1024 || rxErr > 0 || txErr > 0 {
			out.Interfaces = append(out.Interfaces, linuxPerfNetIface{
				Name: name, RxBps: rxBps, TxBps: txBps, RxErr: rxErr, TxErr: txErr,
			})
		}
	}
	return out
}

func parseSockstat() linuxPerfConnections {
	out := linuxPerfConnections{}
	lines, err := readProcLines(procPath("net", "sockstat"))
	if err != nil {
		return out
	}
	for _, ln := range lines {
		f := strings.Fields(ln)
		if len(f) < 3 {
			continue
		}
		switch f[0] {
		case "sockets:":
			if f[1] == "used" {
				out.SocketsUsed, _ = strconv.Atoi(f[2])
			}
		case "TCP:":
			parseSockstatKV(f[1:], &out)
		case "UDP:":
			if f[1] == "inuse" && len(f) >= 3 {
				out.UDPInUse, _ = strconv.Atoi(f[2])
			}
		}
	}
	// ss -s style established from /proc/net/tcp is heavy; use snmp if present
	if est := parseTCPEstablishedFromSNMP(); est > 0 {
		out.TCPEstablished = est
	}
	return out
}

func parseSockstatKV(fields []string, out *linuxPerfConnections) {
	for i := 0; i+1 < len(fields); i += 2 {
		switch fields[i] {
		case "inuse":
			out.TCPInUse, _ = strconv.Atoi(fields[i+1])
		case "orphan":
			out.TCPOrphan, _ = strconv.Atoi(fields[i+1])
		case "tw":
			out.TCPTimeWait, _ = strconv.Atoi(fields[i+1])
		case "alloc":
			out.TCPAlloc, _ = strconv.Atoi(fields[i+1])
		case "embryonic", "synrecv":
			if out.TCPSynRecv == 0 {
				out.TCPSynRecv, _ = strconv.Atoi(fields[i+1])
			}
		}
	}
}

func parseTCPEstablishedFromSNMP() int {
	b, err := readProcFile(procPath("net", "snmp"))
	if err != nil {
		return 0
	}
	lines := strings.Split(string(b), "\n")
	var hdr, val []string
	for _, ln := range lines {
		if !strings.HasPrefix(strings.TrimSpace(ln), "Tcp:") {
			continue
		}
		f := strings.Fields(ln)
		if len(f) < 2 {
			continue
		}
		if strings.Contains(ln, "CurrEstab") {
			hdr = f[1:]
		} else if len(hdr) > 0 && len(val) == 0 {
			val = f[1:]
		}
	}
	for j, h := range hdr {
		if h == "CurrEstab" && j < len(val) {
			n, _ := strconv.Atoi(val[j])
			return n
		}
	}
	return 0
}

func parseTCPStateCounts() (closeWait int) {
	b, err := readProcFile(procPath("net", "tcp"))
	if err != nil {
		return 0
	}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "sl") {
			continue
		}
		f := strings.Fields(ln)
		if len(f) < 4 {
			continue
		}
		// state is hex at field 3 (0A = LISTEN, 01 ESTABLISHED, 06 TIME_WAIT, 08 CLOSE_WAIT)
		st := strings.ToUpper(f[3])
		if st == "08" {
			closeWait++
		}
	}
	return closeWait
}

func collectSystemLimits() linuxPerfSystem {
	var sys linuxPerfSystem
	if b, err := readProcFile(procPath("sys", "fs", "file-nr")); err == nil {
		f := strings.Fields(string(b))
		if len(f) >= 3 {
			sys.OpenFiles, _ = strconv.ParseInt(f[0], 10, 64)
			sys.MaxOpenFiles, _ = strconv.ParseInt(f[2], 10, 64)
		}
	}
	return sys
}
