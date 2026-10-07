package db

import (
	"log"
	"net/http"
)

func StartServer(port string) {
	fs := http.FileServer(http.Dir("db"))
	http.Handle("/db/", http.StripPrefix("/db/", fs))
	http.HandleFunc("/db", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/db/", http.StatusMovedPermanently)
	})

	go func() {
		log.Println("DB server running on port", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Println("DB server error:", err)
		}
	}()
}
