package p

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Photo ...
type Photo struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Image     string  `json:"image"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
}

var testData = `{
  "photos": [
    {
      "id": "photo12",
      "title": "photoTitle1",
      "image": "https://1.bp.blogspot.com/-YnNw0nmy5WY/X5OcdKUoDhI/AAAAAAABb-w/Ws-6a4R4Io4IAWwuxtx8ilCxY9RgmKGHgCNcBGAsYHQ/s180-c/nature_ocean_kaisou.png",
      "latitude": 35.6583865,
      "longitude": 139.7023339
    },
    {
      "id": "photo2",
      "title": "photoTitle2",
      "image": "https://1.bp.blogspot.com/-pc0hWuQtWHA/Xv3UGn3if2I/AAAAAAABZzs/3eWp3hEJZ2AQNtd8gEZf7BsA8xGsfC02gCNcBGAsYHQ/s400/city_kabukichou.png",
      "latitude": 35.658319,
      "longitude": 139.702232
    }
  ]
}`

// MethodGetResponse ...
type MethodGetResponse struct {
	Photos []Photo `json:"photos"`
}

// HTTPFunction is handlerfunc
func HTTPFunction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		res, err := getResponse(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	case http.MethodPost:
		fmt.Fprint(w, "post")
	case http.MethodPatch:
		fmt.Fprint(w, "patch")
	case http.MethodDelete:
		fmt.Fprint(w, "delete")
	}
}

func getResponse(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	resData, err := getTestResponseData()
	if err != nil {
		return []byte{}, err
	}

	res, err := json.Marshal(resData)
	if err != nil {

		return []byte{}, err
	}

	return res, nil
}

func getTestResponseData() (MethodGetResponse, error) {
	var ret MethodGetResponse

	if err := json.Unmarshal([]byte(testData), &ret); err != nil {
		return MethodGetResponse{}, err
	}

	return ret, nil
}
