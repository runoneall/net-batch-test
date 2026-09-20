package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	url       = "https://sin-speed.hetzner.com/10GB.bin"
	batch int = 10
)

var (
	startTime  = make([]atomic.Int64, batch)
	totalBytes = make([]atomic.Uint64, batch)
)

func main() {
	go ui()

	for i := range batch {
		go worker(i)
	}

	select {}
}

func ui() {
	workerIDLength := len(strconv.Itoa(batch))

	for range time.Tick(500 * time.Millisecond) {
		callClear()
		fmt.Println("#", time.Now().Format(time.DateTime))

		var allBytesPerSec float64 = 0
		for i := range batch {
			workerID := fmt.Sprintf("%0*d", workerIDLength, i+1)
			bytesPerSec := calculateSpeed(startTime[i].Load(), totalBytes[i].Load())
			speed := formatSpeed(bytesPerSec)

			fmt.Println("Worker", workerID, "Speed", speed)
			allBytesPerSec += bytesPerSec
		}

		allBytesPerSecAvg := allBytesPerSec / float64(batch)
		speedAvg := formatSpeed(allBytesPerSecAvg)
		fmt.Println("Speed Avg", speedAvg)
	}
}

func worker(i int) {
	for {
		resp, err := http.Get(url)
		if err != nil {
			continue
		}

		startTime[i].Store(time.Now().UnixMilli())
		totalBytes[i].Store(0)

		c := counter{i: i}
		io.Copy(c, resp.Body)

		resp.Body.Close()
	}
}

type counter struct {
	i int
}

func (c counter) Write(p []byte) (int, error) {
	length := len(p)
	totalBytes[c.i].Add(uint64(length))
	return io.Discard.Write(p)
}

func calculateSpeed(startTime int64, totalBytes uint64) float64 {
	elapsed := max(float64(time.Now().UnixMilli()-startTime)/1000.0, 0.001)
	return float64(totalBytes) / elapsed
}

func formatSpeed(bytesPerSec float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}
	u := 0

	for bytesPerSec >= 1024 && u < len(units)-1 {
		bytesPerSec /= 1024
		u++
	}

	return fmt.Sprintf("%15s", fmt.Sprintf("%.1f %s", bytesPerSec, units[u]))
}

func callClear() {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}

	cmd.Stdout = os.Stdout
	cmd.Run()
}
