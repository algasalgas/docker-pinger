package main
import (
	"context"
	"fmt"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"github.com/go-ping/ping"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)
type PingData struct {
	IP          string    `json:"ip"`
	PingTime    float32       `json:"ping_time"`
	LastSuccess time.Time `json:"last_success"`
}

func pingHost(ip string) (float32, error) {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return 0, err
	}
	pinger.Count = 3
	pinger.Timeout = 3 * time.Second
	pinger.SetPrivileged(true)
	if err := pinger.Run(); err != nil {
		return 0, err
	}
	stats := pinger.Statistics()
	return float32(float32(stats.AvgRtt.Microseconds())/1000.), nil
}

func sendPingData(backendURL string, data PingData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := http.Post(backendURL+"/ping-data", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func main() {
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		log.Fatal("BACKEND_URL должен быть задан")
	}
	interval := 5 * time.Second
	if intervalEnv := os.Getenv("PING_INTERVAL"); intervalEnv != "" {
		if d, err := time.ParseDuration(intervalEnv); err == nil {
			interval = d
		}
	}

	log.Printf("Сервис Pinger запущен. Пингуем каждые %s\n", interval)

	for {
		ips, err := getContainerIPs();
		if err != nil {
			log.Printf("не получилось достать IP контейнеров")
		}
		for _, ip := range ips {
			go func(ip string) {
				ip = strings.TrimSpace(ip)
				if ip == "" {
					return
				}
				pingTime, err := pingHost(ip)
				if err != nil {
					log.Printf("Ошибка пинга %s: %v", ip, err)
					return
				}
				log.Println(pingTime)
				data := PingData{
					IP:          ip,
					PingTime:    pingTime,
					LastSuccess: time.Now(),
				}
				err = sendPingData(backendURL, data)
				if err != nil {
					log.Printf("Ошибка отправки данных для %s: %v", ip, err)
				} else {
					log.Printf("Отправлены данные для %s: %f мс", ip, pingTime)
				}
			}(ip)
		}
		time.Sleep(interval)
	}
}
func getContainerIPs() ([]string, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil{
		return nil, fmt.Errorf("не удалось создать клиент Docker: %w", err)
	}
	defer cli.Close()
	containers, err := cli.ContainerList(ctx, containertypes.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список контейнеров: %w", err)
	}
	ips := []string{}
	for _, container := range containers {
		for _, settings := range container.NetworkSettings.Networks{
			if settings.IPAddress != "" {
				ips = append(ips, settings.IPAddress)
			}
		}
	}
	return ips, nil
}