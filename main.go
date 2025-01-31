package main

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type ReceiptResponse struct {
	ID string `json:"id"`
}

type PointsResponse struct {
	Points int `json:"points"`
}

var points = make(map[string]int)

func calculatePoints(receipt Receipt) int {
	totalPoints := 0

	reg := regexp.MustCompile("[a-zA-Z0-9]")
	totalPoints += len(reg.FindAllString(receipt.Retailer, -1))

	if strings.HasSuffix(receipt.Total, ".00") {
		totalPoints += 50
	}

	total, err := strconv.ParseFloat(receipt.Total, 64)
	if err == nil && int(total*100)%25 == 0 {
		totalPoints += 25
	}

	totalPoints += (len(receipt.Items) / 2) * 5

	for _, item := range receipt.Items {
		trimmedDesc := strings.TrimSpace(item.ShortDescription)
		if len(trimmedDesc)%3 == 0 {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err == nil {
				totalPoints += int(price*0.2) + 1
			}
		}
	}

	date, err := time.Parse("2006-01-02", receipt.PurchaseDate)
	if err == nil && date.Day()%2 != 0 {
		totalPoints += 6
	}

	timeParts := strings.Split(receipt.PurchaseTime, ":")
	if len(timeParts) == 2 {
		hour, _ := strconv.Atoi(timeParts[0])
		if hour >= 14 && hour < 16 {
			totalPoints += 10
		}
	}

	return totalPoints
}

func validateReceipt(receipt Receipt) bool {
	patterns := map[string]*regexp.Regexp{
		"retailer":         regexp.MustCompile(`^[\w\s\-&]+$`),
		"total":            regexp.MustCompile(`^\d+\.\d{2}$`),
		"shortDescription": regexp.MustCompile(`^[\w\s\-]+$`),
		"price":            regexp.MustCompile(`^\d+\.\d{2}$`),
	}

	if !patterns["retailer"].MatchString(receipt.Retailer) {
		return false
	}

	if !patterns["total"].MatchString(receipt.Total) {
		return false
	}

	for _, item := range receipt.Items {
		if !patterns["shortDescription"].MatchString(item.ShortDescription) ||
			!patterns["price"].MatchString(item.Price) {
			return false
		}
	}

	return true
}

func processReceipt(c *gin.Context) {
	var receipt Receipt
	if err := c.ShouldBindJSON(&receipt); err != nil || !validateReceipt(receipt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The receipt is invalid."})
		return
	}

	id := uuid.New().String()
	points[id] = calculatePoints(receipt)

	c.JSON(http.StatusOK, ReceiptResponse{ID: id})
}

func getPoints(c *gin.Context) {
	id := c.Param("id")

	if pts, exists := points[id]; exists {
		c.JSON(http.StatusOK, PointsResponse{Points: pts})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "No receipt found for that ID."})
	}
}

func main() {
	router := gin.Default()

	router.POST("/receipts/process", processReceipt)
	router.GET("/receipts/:id/points", getPoints)

	router.Run(":8080")
}
