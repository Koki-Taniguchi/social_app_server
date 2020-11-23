package p

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Photo ...
type Photo struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Image     string  `json:"image"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
}

// ListResponse ...
type ListResponse struct {
	Photos []Photo `json:"photos"`
}

// HTTPFunction is handlerfunc
func HTTPFunction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		res, err := getResponse(w, r)
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	case http.MethodPost:
		uuid.NewUUID()
		fmt.Fprint(w, "post")
	case http.MethodPatch:
		fmt.Fprint(w, "patch")
	case http.MethodDelete:
		fmt.Fprint(w, "delete")
	}
}

func getResponse(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	ctx := r.Context()
	client, err := createClient(ctx)
	if err != nil {
		return []byte{}, wrapError(err)
	}
	defer client.Close()

	iter, err := client.Collection("photos").Documents(ctx).GetAll()
	if err != nil {
		return []byte{}, wrapError(err)
	}

	photos := make([]Photo, 0, len(iter))

	for _, doc := range iter {
		var p Photo
		doc.DataTo(&p)
		photos = append(photos, p)
	}

	res, err := json.Marshal(ListResponse{Photos: photos})
	if err != nil {
		return []byte{}, wrapError(err)
	}

	return res, nil
}

func createClient(ctx context.Context) (*firestore.Client, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return &firestore.Client{}, wrapError(err)
	}

	return client, nil
}

func wrapError(err error) error {
	pc, _, line, _ := runtime.Caller(1)
	f := runtime.FuncForPC(pc)
	message := fmt.Sprintf("\nerror in %s method. line: %d", f.Name(), line)
	return errors.Wrap(err, message)
}
