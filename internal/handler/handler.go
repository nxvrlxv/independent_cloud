package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/nxvrlxv/independent_cloud/internal/auth"
	"github.com/nxvrlxv/independent_cloud/internal/repository"
	"github.com/nxvrlxv/independent_cloud/internal/service"
)

type FileHandler struct {
	serv *service.FileService
	auth *auth.AuthService
}

func NewFileHandler(s *service.FileService, auth *auth.AuthService) *FileHandler {
	return &FileHandler{serv: s, auth: auth}
}

// func (s *FileHandler) FirstHandler(w http.ResponseWriter, r *http.Request) {
// 	// ctx := r.Context()
// 	// r.FormFile()
// 	fmt.Fprintf(w, "Урааа первый эндпоинт %s", r.URL.Path)
// }

func (s *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err != nil {
		http.Error(w, "there is no file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	var FolderID *int64
	FolderIDStr := r.FormValue("folder_id")
	if FolderIDStr != "" {
		parsed, err := strconv.Atoi(FolderIDStr)
		if err != nil {
			http.Error(w, "Incorrect folder id!", http.StatusBadRequest)
			return
		}
		p64 := int64(parsed)
		FolderID = &p64
	}

	ctx := r.Context()
	id, err := s.serv.Upload(ctx, file, userID, header.Filename, header.Header.Get("Content-Type"), FolderID)
	if err != nil {
		fmt.Println(err)
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
		return
	}
}

func (s *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	file_id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "failed to found data", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	reader, file, err := s.serv.Download(ctx, int64(file_id), userID)
	if err != nil {
		http.Error(w, "failed to download file", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.OriginalName))
	_, err1 := io.Copy(w, reader)
	if err1 != nil {
		http.Error(w, "something went wrong when streaming", http.StatusInternalServerError)
		return
	}

	type response struct {
		Filename string `json:"filename"`
		Message  string `json:"msg"`
	}
	err2 := json.NewEncoder(w).Encode(response{Filename: file.OriginalName, Message: "file succsessfully downloaded!"})
	if err2 != nil {
		http.Error(w, "error occured while foramtting answer", http.StatusNoContent)
		return
	}

}

func (s *FileHandler) Update(w http.ResponseWriter, r *http.Request) {
	type req struct {
		NewFileName string `json:"new_name"`
		NewFolderID *int64 `json:"new_folder_id"`
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
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if reqName.NewFileName != "" {
		err1 := s.serv.Update(ctx, int64(fileId), reqName.NewFileName, userID)

		if err1 != nil {
			http.Error(w, "Error occured while updating data", http.StatusInternalServerError)
			return
		}
	}

	if err := s.serv.ChangeFolder(ctx, int64(fileId), reqName.NewFolderID, userID); err != nil {
		http.Error(w, "some error in changing Folder for file", http.StatusInternalServerError)
		return
	}
}

func (s *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	fileID, err := strconv.Atoi(r.PathValue("id"))
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "don't have file with this id", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	err1 := s.serv.Delete(ctx, int64(fileID), userID)
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

type requestFolders struct {
	FolderName string `json:"folder_name"`
	ParentID   *int64 `json:"parent_id"`
}

// эндпоинты для папок
func (s *FileHandler) AddFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Error:", http.StatusMethodNotAllowed)
		return
	}
	req := requestFolders{}

	ctx := r.Context()
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Incorrect data:", http.StatusBadRequest)
		return
	}

	FolderID, err1 := s.serv.AddFolder(ctx, userID, req.FolderName, req.ParentID)
	if err1 != nil {
		http.Error(w, "Can't create a folder", http.StatusInternalServerError)
		return
	}
	type response struct {
		FolderID int64 `json:"folder_id"`
	}
	res := response{FolderID: FolderID}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&res); err != nil {
		http.Error(w, "invalid parameters in JSON", http.StatusBadRequest)
		return
	}

}

func (s *FileHandler) ListFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Error:", http.StatusMethodNotAllowed)
		return
	}
	type response struct {
		Folders []repository.Folder `json:"folders"`
	}

	ctx := r.Context()
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	folders, err := s.serv.ListOfFolders(ctx, userID)
	if err != nil {
		http.Error(w, "error getting Folders", http.StatusInternalServerError)
		return
	}
	if folders == nil {
		folders = []repository.Folder{}
	}
	res := response{Folders: folders}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err1 := json.NewEncoder(w).Encode(&res)
	if err1 != nil {
		http.Error(w, "Invalid data", http.StatusInternalServerError)
		return
	}

}

func (s *FileHandler) ListUserFiles(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileList, err := s.serv.List(r.Context(), userID)
	if err != nil {
		http.Error(w, "error getting files", http.StatusInternalServerError)
		return
	}

	if fileList == nil {
		fileList = []repository.File{}
	}

	type response struct {
		Files []repository.File `json:"files"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response{Files: fileList}); err != nil {
		http.Error(w, "Invalid data", http.StatusInternalServerError)
		return
	}
}

func (s *FileHandler) ListOfContentsInFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Error:", http.StatusMethodNotAllowed)
		return
	}

	FolderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id!", http.StatusBadRequest)
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ctx := r.Context()
	content, err1 := s.serv.ListOfContentsInFolder(ctx, int64(FolderID), userID)
	if err1 != nil {
		http.Error(w, "Error in list of files", http.StatusExpectationFailed)
		return
	}
	if content == nil {
		content = &repository.ContentInFolder{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	err2 := json.NewEncoder(w).Encode(content)
	if err2 != nil {
		http.Error(w, "Error in files to JSON", http.StatusInternalServerError)
		return
	}

}

func (s *FileHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	FolderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "wrong ID", http.StatusBadRequest)
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := s.serv.DeleteFolder(ctx, int64(FolderID), userID); err != nil {
		http.Error(w, "Error occured while deleting folder", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

}

func (s *FileHandler) RenameFolder(w http.ResponseWriter, r *http.Request) {

	type request struct {
		NewFolderName string `json:"new_folder_name"`
	}
	req := request{}
	ctx := r.Context()
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	FolderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "wrong ID", http.StatusBadRequest)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid data from JSON", http.StatusBadRequest)
		return
	}

	if err := s.serv.RenameFolder(ctx, int64(FolderID), req.NewFolderName, userID); err != nil {
		http.Error(w, "errorr", http.StatusInternalServerError)
		return
	}

}

type request struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type response struct {
	UserID int64 `json:"user_id"`
}

func (s *FileHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {

	req := request{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid parameters JSON", http.StatusBadRequest)
		return
	}

	id, err := s.auth.RegisterUser(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, "Error in registration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err1 := json.NewEncoder(w).Encode(response{UserID: id})
	if err1 != nil {
		http.Error(w, "Error occured while decode to json", http.StatusInternalServerError)
		return
	}

}

func (s *FileHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	req := request{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid parameters JSON", http.StatusBadRequest)
		return
	}

	id, err := s.auth.LoginUser(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	tokenString, err := s.auth.GenerateToken(id)
	if err != nil {
		http.Error(w, "Error in creating your token", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err1 := json.NewEncoder(w).Encode(response{UserID: id})
	if err1 != nil {
		http.Error(w, "Json", http.StatusInternalServerError)
		return
	}

}

func (s *FileHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
}
