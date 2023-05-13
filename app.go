package main

import (
	"context"
	"database/sql"
	"fmt"
)

// App struct
type App struct {
	ctx context.Context
	db  *sql.DB
}

type Embedding struct {
	Embedding []float32 `json:"embedding"`
	Name      string    `json:"name"`
	Length    int       `json:"length"`
}

// NewApp creates a new App application struct
func NewApp(db *sql.DB) *App {
	return &App{db: db}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	// Perform your setup here
	a.ctx = ctx
}

// domReady is called after front-end resources have been loaded
func (a App) domReady(ctx context.Context) {
	// Add your action here
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	// Perform your teardown here
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) Test() string {
	return "Hello World!"
}

type Test struct {
	Id  int   `json:"id"`
	Name string `json:"name"`
}

func (a *App) GetTest() ([]Test, error) {
	rows, err := a.db.Query("SELECT * FROM test")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()

	var data []Test
	for rows.Next() {
		var res Test
		err := rows.Scan(&res.Id, &res.Name)
		if err != nil {
			return nil, err
		}
		data = append(data, res)
	}

	return data, nil
}

func (a *App) Struct() Embedding {
	return Embedding{
		Embedding: []float32{1.0, 2.0, 3.0},
		Name:      "test",
		Length:    3,
	}
}
