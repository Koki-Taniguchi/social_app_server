package p

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	// HectometerLat is Latitude per Hectometer in Japan
	HectometerLat = 0.0009013
	// HectometerLng is Longitude per Hectometer in Japan
	HectometerLng = 0.0010966
)

// Photo ...
type Photo struct {
	ID        string  `json:"id" validate:"required"`
	Title     string  `json:"title" validate:"required"`
	Image     string  `json:"image" validate:"required"`
	Latitude  float32 `json:"latitude" validate:"required"`
	Longitude float32 `json:"longitude" validate:"required"`
}

// ListResponse ...
type ListResponse struct {
	Photos []Photo `json:"photos"`
}

// HTTPFunction is handlerfunc
func HTTPFunction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		// GET: /photos
		lat, err := strconv.ParseFloat(r.FormValue("lat"), 64)
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		lng, err := strconv.ParseFloat(r.FormValue("lng"), 64)
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := getPhotoList(ctx, lat, lng)
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	case http.MethodPost:
		// Post: /photos
		photo, err := convertToPhoto(r)
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := uploadFile(ctx, &photo); err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err = createPhoto(ctx, photo); err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("message: created successfully!"))
	case http.MethodPatch:
		// Patch: /photos
		photo, err := convertToPhoto(r)
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err = updatePhoto(ctx, photo); err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("message: updated successfully!"))
	case http.MethodDelete:
		// Delete: /photos?id="id"
		if err := deletePhoto(ctx, r.FormValue("id")); err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("message: deleted successfully!"))
	}
}

func getPhotoList(ctx context.Context, lat, lng float64) ([]byte, error) {
	client, err := firestoreClient(ctx)
	if err != nil {
		return nil, wrapError(err)
	}
	defer client.Close()

	iter, err := client.Collection("photos").Documents(ctx).GetAll()
	if err != nil {
		return nil, wrapError(err)
	}

	photos := make([]Photo, 0, len(iter))

	// firestoreは多重queryを利用できないので全件取得して絞り込む
	for _, doc := range iter {
		var p Photo
		if err := doc.DataTo(&p); err != nil || !includeInArea(p, float32(lat), float32(lng)) {
			continue
		}

		photos = append(photos, p)
	}

	res, err := json.Marshal(ListResponse{Photos: photos})
	if err != nil {
		return nil, wrapError(err)
	}

	return res, nil
}

func createPhoto(ctx context.Context, photo Photo) error {
	client, err := firestoreClient(ctx)
	if err != nil {
		return wrapError(err)
	}
	defer client.Close()

	if _, err := client.Collection("photos").Doc(photo.ID).Set(ctx, photo); err != nil {
		return wrapError(err)
	}

	return nil
}

func updatePhoto(ctx context.Context, photo Photo) error {
	client, err := firestoreClient(ctx)
	if err != nil {
		return wrapError(err)
	}
	defer client.Close()

	targetDoc := client.Collection("photos").Doc(photo.ID)

	dsnap, err := targetDoc.Get(ctx)
	if err != nil {
		return wrapError(err)
	}

	if !dsnap.Exists() {
		return errors.New(fmt.Sprintf("%s is empty.", photo.ID))
	}

	if _, err := targetDoc.Update(ctx, []firestore.Update{
		{Path: "Title", Value: photo.Title},
		{Path: "Image", Value: photo.Image},
		{Path: "Latitude", Value: photo.Latitude},
		{Path: "Longitude", Value: photo.Longitude},
	}); err != nil {
		return wrapError(err)
	}

	return nil
}

func deletePhoto(ctx context.Context, photoID string) error {
	client, err := firestoreClient(ctx)
	if err != nil {
		return wrapError(err)
	}
	defer client.Close()

	targetDoc := client.Collection("photos").Doc(photoID)

	dsnap, err := targetDoc.Get(ctx)
	if err != nil {
		return wrapError(err)
	}

	if !dsnap.Exists() {
		return errors.New(fmt.Sprintf("%s is empty.", photoID))
	}

	if _, err := targetDoc.Delete(ctx); err != nil {
		return wrapError(err)
	}

	return nil
}

func includeInArea(photo Photo, lat, lng float32) bool {
	checkLat := photo.Latitude <= lat+HectometerLat && photo.Latitude >= lat-HectometerLat
	checkLng := photo.Longitude <= lng+HectometerLng && photo.Longitude >= lng-HectometerLng

	return checkLat && checkLng
}

func convertToPhoto(r *http.Request) (Photo, error) {
	reader, err := gzip.NewReader(r.Body)
	if err != nil {
		return Photo{}, wrapError(err)
	}
	var unzipByte []byte

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		unzipByte = scanner.Bytes()
	}
	var photo Photo

	if err := json.Unmarshal(unzipByte, &photo); err != nil {
		return Photo{}, wrapError(err)
	}

	if photo.ID == "" {
		_uuid, err := uuid.NewUUID()
		if err != nil {
			return Photo{}, wrapError(err)
		}
		photo.ID = _uuid.String()
	}

	validate := validator.New()
	if err := validate.Struct(photo); err != nil {
		return Photo{}, wrapError(err)
	}

	return photo, nil
}

func uploadFile(ctx context.Context, photo *Photo) error {
	data, _ := base64.RawStdEncoding.DecodeString(photo.Image)

	client, err := storage.NewClient(ctx)
	if err != nil {
		return wrapError(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	wc := client.Bucket(os.Getenv("BUCKET_NAME")).Object(photo.ID + ".png").NewWriter(ctx)
	wc.ContentType = "image/png"

	if _, err := wc.Write(data); err != nil {
		return wrapError(err)
	}
	if err := wc.Close(); err != nil {
		return wrapError(err)
	}

	photo.Image = fmt.Sprintf("https://storage.googleapis.com/%s/%s.png", os.Getenv("BUCKET_NAME"), photo.ID)

	return nil
}

func firestoreClient(ctx context.Context) (*firestore.Client, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, wrapError(err)
	}

	return client, nil
}

func wrapError(err error) error {
	pc, _, line, _ := runtime.Caller(1)
	f := runtime.FuncForPC(pc)
	message := fmt.Sprintf("\nerror in %s method. line: %d", f.Name(), line)
	return errors.Wrap(err, message)
}
