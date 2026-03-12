package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/lexxcode1/yop-pms/internal/platform/config"
)

func main() {
	cfg := config.MustLoad()
	url := cfg.DatabaseURL

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		fmt.Printf("Connection failed: %v\nCheck if your password is correct or if the port is 5432.\n", err)
		return
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `CREATE ROLE app_user WITH LOGIN PASSWORD 'password';`)
	if err != nil {
		fmt.Printf("Role error: %v (It might already exist)\n", err)
	}

	_, err = conn.Exec(ctx, `
		GRANT ALL PRIVILEGES ON SCHEMA operations, inventory, pricing, finance, sales_ledgers, identity, auth, relations TO app_user;
		GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA operations, inventory, pricing, finance, sales_ledgers, identity, auth, relations TO app_user;
		GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA operations, inventory, pricing, finance, sales_ledgers, identity, auth, relations TO app_user;
	`)

	if err != nil {
		fmt.Printf("Permission error: %v\n", err)
	} else {
		fmt.Println("✅ app_user created and permissions granted!")
	}
}
