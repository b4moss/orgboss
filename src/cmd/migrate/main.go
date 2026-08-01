package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/b4m-oss/orgboss/internal/database"
	"github.com/b4m-oss/orgboss/internal/seed"
)

func main() {
	seedFlag := flag.Bool("seed", false, "Seed the database with test data")
	flag.Parse()

	// データベースに接続
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}
	defer sqlDB.Close()

	// マイグレーション実行
	fmt.Println("Running migrations...")
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	fmt.Println("Migrations completed successfully")

	// シーディングが指定されている場合
	if *seedFlag {
		fmt.Println("Seeding database...")
		ctx := context.Background()
		seedData, err := seed.Seed(ctx, db)
		if err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
		fmt.Printf("Seeded %d organizations, %d users, %d invitations\n",
			len(seedData.Organizations), len(seedData.Users), len(seedData.Invitations))
		fmt.Println("Seeding completed successfully")
	}
}
