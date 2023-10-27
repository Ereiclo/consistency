package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Data struct {
	ItemID   uint `gorm:"primary_key"`
	Name     string
	Quantity int `gorm:"check:Quantity>=0"`
}

type BuyData struct {
	ItemID   uint `json:"ItemId"`
	Quantity int  `json:"Quantity"`
}

func main() {

	// Create flag PORT
	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dsn := os.Getenv("DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&Data{})

	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))
	e.GET("/query", func(c echo.Context) error {
		// obtain query parameters
		qp_itemId := c.QueryParam("ItemId")
		qp_type := c.QueryParam("type")

		if qp_itemId != "" {
			// query database
			var data Data
			r := db.First(&data, qp_itemId)
			if r.Error != nil {
				return c.JSON(http.StatusBadRequest, "No such item")
			} // returns error if any
			// Convert to json
			return c.JSON(http.StatusOK, data)
		} else if qp_type == "ALL" {
			// query database for all rows
			var data []Data
			db.Find(&data)
			// Convert to json
			return c.JSON(http.StatusOK, data)
		} else {
			return c.JSON(http.StatusBadRequest, "No query parameters")
		}

	})
	e.GET("/buy", func(c echo.Context) error {
		// Get json body
		var buyData BuyData
		if err := c.Bind(&buyData); err != nil {
			return c.JSON(http.StatusBadRequest, "No json body")
		}
		// query database
		var data Data
		r := db.First(&data, buyData.ItemID)
		if r.Error != nil {
			return c.JSON(http.StatusBadRequest, "No such item")
		}
		// check if enough quantity
		if data.Quantity < buyData.Quantity {
			return c.JSON(http.StatusBadRequest, "Not enough quantity")
		}
		// update database

		err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&data).UpdateColumn("Quantity", gorm.Expr("Quantity - ?", buyData.Quantity)).Error

		log.Println(err)
		//make a log of the error

		if err != nil {
			return c.JSON(http.StatusBadRequest, "Not enough quantity")
		}

		// return c.JSON(http.StatusBadRequest, "Quantity too low")
		// }
		// Convert to json
		return c.JSON(http.StatusOK, data)
	})
	e.Logger.Fatal(e.Start(":" + *port))
}
