package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	// undescore (alias) is used to avoid go compiler complaining or erasing this
	// library.
	_ "github.com/lib/pq"
	"github.com/shynggys9219/greenlight/internal/data"
	"github.com/shynggys9219/greenlight/internal/mailer"
)

const version = "1.0.0"

// Add a db struct field to hold the configuration settings for our database connection
// pool. For now this only holds the DSN, which we will read in from a command-line flag.
type ConfTest struct {
	Port int
}
type Config struct {
	port int
	env  string
	db   struct {
		dsn          string // a conenction string to a sql server
		maxOpenConns int    // limit on the number of ‘open’ connections
		maxIdleConns int    // limit on the number of idle connections in the pool
		maxIdleTime  string // the maximum length of time that a connection can be idle
		// maxLifetime  string //optional here; maximum length of time that a connection can be reused for
	}
	limiter struct {
		enabled bool
		rps     float64
		burst   int
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	config Config
	logger *log.Logger
	models data.Models // hold new models in app
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	var cfg Config
	flag.IntVar(&cfg.port, "port", 9000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	// Read the DSN value from the db-dsn command-line flag into the config struct. We
	// default to using our development DSN if no flag is provided.
	// in powershell use next command: $env:DSN="postgres://postgres:postgres@localhost:5432/greenlight?sslmode=disable"
	flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://postgres:postgres@localhost:5432/aitu_golang?sslmode=disable", "PostgreSQL DSN")

	// Setting restrictions on db connections
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max idle time")
	// flag.StringVar(&cfg.db.maxLifetime, "db-max-lifetime", "1h", "PostgreSQL max idle time")

	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "0abf276416b183", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "d8672aa2264bb5", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")

	flag.Parse()
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := OpenDB(cfg)
	if err != nil {
		logger.Fatalf("Connection failed. Error is: %s", err)
	}
	// db will be closed before main function is completed.
	defer db.Close()
	logger.Printf("database connection pool established")

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db), // data.NewModels() function to initialize a Models struct
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	// Use the httprouter instance returned by app.routes() as the server handler.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)
	// reuse defined variable err
	err = srv.ListenAndServe()
	logger.Fatal(err)
}

func OpenDB(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	// optional lifetime limit, to use this, uncomment db substruct field and corresponding flag stringvar
	// lifetime, err := time.ParseDuration(cfg.db.maxIdleTime)
	// if err != nil {
	// 	return nil, err
	// }
	// db.SetConnMaxLifetime(lifetime)

	//context with a 5 second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx) //create a connection and verify that everything is set up correctly.

	if err != nil {
		return nil, err
	}

	return db, nil
}
