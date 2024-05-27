// Code generated by github.com/mohuishou/protoc-gen-go-gin. DO NOT EDIT.

package testv1

import (
	context "context"
	json "encoding/json"
	errors "errors"
	io "io"
	http "net/http"
)

// TestServiceServer is the server API for TestService service.
type TestServiceServer interface {
	GameLaunch(context.Context, *GameLaunchInput) (*GameLaunchResult, error)
}

func writeErr(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(`{"error":` + err.Error() + `}`))
}

func writeRsp(w http.ResponseWriter, resp any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	respba, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(respba)
}
func RegisterHttpServer(srv any, impl TestServiceServer) (err error) {
	mux, ok := srv.(interface{ Handle(string, http.Handler) })
	if !ok {
		err = errors.New("srv must implement HttpServerMux")
		return
	}
	mux.Handle(GameLaunchHandler(impl))
	return
}

// GameLaunch returns TestServiceHTTPService interface's GameLaunch converted to http.HandlerFunc.
func GameLaunchHandler(srv TestServiceServer) (pattern string, hdr http.Handler) {
	pattern = "POST /api/v1/gamelaunch/{id}"
	hdr = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		in := &GameLaunchInput{}
		if r.Method != http.MethodGet {
			reqba, err := io.ReadAll(r.Body)
			if err != nil {
				writeErr(w, err)
				return
			}
			err = json.Unmarshal(reqba, in)
			if err != nil {
				writeErr(w, err)
				return
			}
		}
		in.Id = r.PathValue("id")
		out, err := srv.GameLaunch(ctx, in)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeRsp(w, out)
	})
	return
}