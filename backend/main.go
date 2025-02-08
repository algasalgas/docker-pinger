package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type ContainerStatus struct {
  ID int `json:"id"`
  IP          string    `json:"ip"`
	PingTime    float32       `json:"ping_time"` 
	LastSuccess time.Time `json:"last_success"`
}
var db *sql.DB
func main(){
  dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")

	connStr := "postgres://" + dbUser + ":" + dbPass + "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable"
	var err error
  db, err = sql.Open("postgres", connStr)
  if err != nil{
    log.Fatal("Ошибка подключения к БД:", err)
  }
  defer db.Close()
  _, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS container_status (
      id SERIAL PRIMARY KEY,
      ip TEXT NOT NULL,
      ping_time DOUBLE PRECISION,
      last_success TIMESTAMP
    )
  `)
  if err != nil {
    log.Fatal("Ошибка создания таблицы:", err)
  }
	router := gin.Default()
	router.GET("/ping-data", getPingData)
	router.POST("/ping-data", addPingData)

	log.Println("Backend запущен на порту 8080")
	router.Run(":8080")
}
func getPingData(c *gin.Context) {
	rows, err := db.Query(`
    SELECT DISTINCT ON (ip) id, ip, ping_time, last_success
    FROM container_status
    ORDER BY ip, last_success DESC
  `)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var statuses []ContainerStatus
	for rows.Next() {
		var s ContainerStatus
		if err := rows.Scan(&s.ID, &s.IP, &s.PingTime, &s.LastSuccess); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		statuses = append(statuses, s)
	}
	c.JSON(http.StatusOK, statuses)
}
func addPingData(c *gin.Context) {
	var s ContainerStatus
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.LastSuccess.IsZero() {
		s.LastSuccess = time.Now()
	}
	err := db.QueryRow(
		"INSERT INTO container_status (ip, ping_time, last_success) VALUES ($1, $2, $3) RETURNING id",
		s.IP, s.PingTime, s.LastSuccess,
	).Scan(&s.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, s)
}