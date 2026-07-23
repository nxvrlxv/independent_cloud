package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/nxvrlxv/independent_cloud/internal/repository"
	"github.com/nxvrlxv/independent_cloud/internal/service"
)

type FileHandler struct {
	serv *service.FileService
}

func NewFileHandler(s *service.FileService) *FileHandler {
	return &FileHandler{serv: s}
}

// func (s *FileHandler) FirstHandler(w http.ResponseWriter, r *http.Request) {
// 	// ctx := r.Context()
// 	// r.FormFile()
// 	fmt.Fprintf(w, "Урааа первый эндпоинт %s", r.URL.Path)
// }

func (s *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "there is no file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ctx := r.Context()
	id, err := s.serv.Upload(ctx, file, 1, header.Filename, header.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, "error occured while uploading file", http.StatusInternalServerError)
		return
	}
	type response struct {
		ID int64 `json:"id"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err1 := json.NewEncoder(w).Encode(response{ID: id})
	if err1 != nil {
		http.Error(w, "error occured while foramtting answer", http.StatusInternalServerError)
	}
}

func (s *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	file_id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "failed to found data", http.StatusBadRequest)
	}
	ctx := r.Context()

	reader, file, err := s.serv.Download(ctx, int64(file_id), 1)
	if err != nil {
		http.Error(w, "failed to download file", http.StatusInternalServerError)
	}
	defer reader.Close()

	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.OriginalName))
	_, err1 := io.Copy(w, reader)
	if err1 != nil {
		http.Error(w, "something went wrong when streaming", http.StatusInternalServerError)
	}

	type response struct {
		Filename string `json:"filename"`
		Message  string `json:"msg"`
	}
	err2 := json.NewEncoder(w).Encode(response{Filename: file.OriginalName, Message: "file succsessfully downloaded!"})
	if err2 != nil {
		http.Error(w, "error occured while foramtting answer", http.StatusNoContent)
	}

}

func (s *FileHandler) Update(w http.ResponseWriter, r *http.Request) {
	type req struct {
		NewFileName string `json:"new_name"`
	}

	reqName := req{}

	err := json.NewDecoder(r.Body).Decode(&reqName)
	if err != nil {
		fmt.Printf("decode error: %v\n", err)
		http.Error(w, "wrong parameters", http.StatusBadRequest)
		return
	}
	fmt.Println(reqName.NewFileName)
	// NewFilename := r.URL.Query().Get("newFilename")
	// fmt.Println(NewFilename)
	fileId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Error while getting path parameter", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	err1 := s.serv.Update(ctx, int64(fileId), reqName.NewFileName)
	if err1 != nil {
		http.Error(w, "Error occured while updating data", http.StatusInternalServerError)
		return
	}
}

func (s *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	fileID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "don't have file with this id", http.StatusBadRequest)
	}
	ctx := r.Context()
	err1 := s.serv.Delete(ctx, int64(fileID), 1)
	if err1 != nil {
		switch {
		case errors.Is(err, repository.ErrFileNotFound):
			http.Error(w, "file not found", http.StatusNotFound)
		case errors.Is(err, service.ErrAccessDenied):
			http.Error(w, "access denied", http.StatusForbidden)
		default:
			http.Error(w, "failed to delete file", http.StatusInternalServerError)
		}
		return
	}

}

//эндпоинты для папок
