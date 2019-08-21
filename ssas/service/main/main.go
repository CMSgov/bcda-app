package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service"
	"github.com/CMSgov/bcda-app/ssas/service/admin"
	"github.com/CMSgov/bcda-app/ssas/service/public"
)

var startMeUp bool
var migrateAndStart bool

func init() {
	const usageStart = "start the service"
	flag.BoolVar(&startMeUp, "start", false, usageStart)
	flag.BoolVar(&startMeUp, "s", false, usageStart+" (shorthand)")
	const usageMigrate = "migrate the db and start the service"
	flag.BoolVar(&migrateAndStart, "migrate", false, usageMigrate)
	flag.BoolVar(&migrateAndStart, "m", false, usageMigrate+" (shorthand)")
}

func main() {
	ssas.Logger.Info("Future home of the System-to-System Authentication Service")
	flag.Parse()
	if migrateAndStart {
		if os.Getenv("DEBUG") == "true" {
			ssas.InitializeGroupModels()
			ssas.InitializeSystemModels()
			ssas.InitializeBlacklistModels()
			addFixtureData()
		}
		start()
	}
	if startMeUp {
		start()
	}
}

func start() {
	ssas.Logger.Infof("%s", "Starting ssas...")

	ps := public.Server()
	if ps == nil {
		ssas.Logger.Error("unable to create public server")
		os.Exit(-1)
	}
	ps.LogRoutes()
	ps.Serve()

	as := admin.Server()
	if as == nil {
		ssas.Logger.Error("unable to create admin server")
		os.Exit(-1)
	}
	as.LogRoutes()
	as.Serve()

	// Accepts and redirects HTTP requests to HTTPS. Not sure we should do this.
	forwarder := &http.Server{
		Handler:      newForwardingRouter(),
		Addr:         ":3005",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	ssas.Logger.Fatal(forwarder.ListenAndServe())
}

func newForwardingRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(service.NewAPILogger(), service.ConnectionClose)
	r.Get("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO only forward requests for paths in our own host or resource server
		url := "https://" + req.Host + req.URL.String()
		ssas.Logger.Infof("forwarding from %s to %s", req.Host+req.URL.String(), url)
		http.Redirect(w, req, url, http.StatusMovedPermanently)
	}))
	return r
}

func addFixtureData() {
	db := ssas.GetGORMDbConnection()
	defer ssas.Close(db)

	if err := db.Save(&ssas.Group{GroupID: "admin"}).Error; err != nil {
		fmt.Println(err)
	}
	// group for cms_id A9994; aco_id 0c527d2e-2e8a-4808-b11d-0fa06baf8254
	if err := db.Save(&ssas.Group{GroupID: "0c527d2e-2e8a-4808-b11d-0fa06baf8254"}).Error; err != nil {
		fmt.Println(err)
	}
	makeSystem(db, "admin", "31e029ef-0e97-47f8-873c-0e8b7e7f99bf",
		"BCDA API Admin", "bcda-admin",
		"nbZ5oAnTlzyzeep46bL4qDGGuidXuYxs3xknVWBKjTI=:9s/Tnqvs8M7GN6VjGkLhCgjmS59r6TaVguos8dKV9lGqC1gVG8ywZVEpDMkdwOaj8GoNe4TU3jS+OZsK3kTfEQ==",
	)
	makeSystem(db, "0c527d2e-2e8a-4808-b11d-0fa06baf8254",
		"0c527d2e-2e8a-4808-b11d-0fa06baf8254", "ACO Dev", "bcda-api",
		"h5hF9cm0Wmm+ClnoF0+Dq5JCQFmDVtzAsaquigoYcTk=:mptcWsBLNYFylRT1q95brbfKiaQkUt8oZXml0EMXobghbVVewZeG40EfNqe10u1+RftftEMvzSCB/oG17MRpVA==")
}

func makeSystem(db *gorm.DB, groupID, clientID, clientName, scope, hash string) {
	pem := `-----BEGIN PUBLIC KEY-----
	MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArhxobShmNifzW3xznB+L
	I8+hgaePpSGIFCtFz2IXGU6EMLdeufhADaGPLft9xjwdN1ts276iXQiaChKPA2CK
	/CBpuKcnU3LhU8JEi7u/db7J4lJlh6evjdKVKlMuhPcljnIKAiGcWln3zwYrFCeL
	cN0aTOt4xnQpm8OqHawJ18y0WhsWT+hf1DeBDWvdfRuAPlfuVtl3KkrNYn1yqCgQ
	lT6v/WyzptJhSR1jxdR7XLOhDGTZUzlHXh2bM7sav2n1+sLsuCkzTJqWZ8K7k7cI
	XK354CNpCdyRYUAUvr4rORIAUmcIFjaR3J4y/Dh2JIyDToOHg7vjpCtNnNoS+ON2
	HwIDAQAB
	-----END PUBLIC KEY-----`
	system := ssas.System{GroupID: groupID, ClientID: clientID, ClientName: clientName, APIScope: scope}
	if err := db.Save(&system).Error; err != nil {
		ssas.Logger.Warn(err)
	}
	encryptionKey := ssas.EncryptionKey{
		Body:     pem,
		SystemID: system.ID,
	}
	if err := db.Save(&encryptionKey).Error; err != nil {
		ssas.Logger.Warn(err)
	}
	secret := ssas.Secret{
		Hash:     hash,
		SystemID: system.ID,
	}
	if err := db.Save(&secret).Error; err != nil {
		ssas.Logger.Warn(err)
	}
}
