package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type TodoStorage interface {
	Save(w io.Writer) error
	Load(r io.Reader) error
	Create(todo *Todo) error
	Read(id int) (*Todo, error)
	Update(id int, updates map[string]interface{}) error
	Delete(id int) error
	List() ([]*Todo, error)
	SaveFile(todoID int, fileName string, r io.Reader, size int64) error
	LoadFile(todoID int, fileName string, w io.Writer) error
}

type FileStorage struct {
	mu        sync.RWMutex
	todos     map[int]*Todo
	nextID    int
	dataDir   string
	todosFile string
}

func NewFileStorage(dataDir string) *FileStorage {
	return &FileStorage{
		todos:     make(map[int]*Todo),
		nextID:    1,
		dataDir:   dataDir,
		todosFile: filepath.Join(dataDir, "todos.json"),
	}
}

func (fs *FileStorage) EnsureDataDir() error {
	return os.MkdirAll(fs.dataDir, 0755)
}

func (fs *FileStorage) Save(w io.Writer) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	bw := bufio.NewWriter(w)
	defer bw.Flush()

	encoder := json.NewEncoder(bw)
	encoder.SetIndent("", "  ")

	data := struct {
		NextID int           `json:"next_id"`
		Todos  map[int]*Todo `json:"todos"`
	}{
		NextID: fs.nextID,
		Todos:  fs.todos,
	}

	return encoder.Encode(data)
}

func (fs *FileStorage) Load(r io.Reader) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	br := bufio.NewReader(r)
	decoder := json.NewDecoder(br)

	var data struct {
		NextID int           `json:"next_id"`
		Todos  map[int]*Todo `json:"todos"`
	}

	if err := decoder.Decode(&data); err != nil {
		return err
	}

	fs.nextID = data.NextID
	fs.todos = data.Todos
	return nil
}

func (fs *FileStorage) Create(todo *Todo) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	todo.Id = fs.nextID
	fs.nextID++
	if todo.Files == nil {
		todo.Files = make([]*File, 0)
	}
	fs.todos[todo.Id] = todo
	return nil
}

func (fs *FileStorage) Read(id int) (*Todo, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	todo, exists := fs.todos[id]
	if !exists {
		return nil, fmt.Errorf("todo %d not found", id)
	}
	return todo, nil
}

func (fs *FileStorage) Update(id int, updates map[string]interface{}) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	todo, exists := fs.todos[id]
	if !exists {
		return fmt.Errorf("todo %d not found", id)
	}

	if title, ok := updates["title"].(string); ok && title != "" {
		todo.Title = title
	}
	if done, ok := updates["done"].(bool); ok {
		todo.Done = done
	}

	return nil
}

func (fs *FileStorage) Delete(id int) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.todos[id]; !exists {
		return fmt.Errorf("todo %d not found", id)
	}

	delete(fs.todos, id)
	return nil
}

func (fs *FileStorage) List() ([]*Todo, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	todos := make([]*Todo, 0, len(fs.todos))
	for _, todo := range fs.todos {
		todos = append(todos, todo)
	}
	return todos, nil
}

func (fs *FileStorage) SaveFile(todoID int, fileName string, r io.Reader, size int64) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	todo, exists := fs.todos[todoID]
	if !exists {
		return fmt.Errorf("todo %d not found", todoID)
	}

	fileDir := filepath.Join(fs.dataDir, "files", fmt.Sprintf("todo_%d", todoID))
	os.MkdirAll(fileDir, 0755)

	filePath := filepath.Join(fileDir, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	bw := bufio.NewWriter(file)
	defer bw.Flush()

	lr := io.LimitReader(r, size)
	written, err := io.Copy(bw, lr)
	if err != nil {
		return err
	}

	fileRef := &File{
		Path: fileName,
		Data: nil,
	}

	todo.Files = append(todo.Files, fileRef)

	fmt.Printf("Saved file %s (%d bytes) for todo %d\n", fileName, written, todoID)
	return nil
}

func (fs *FileStorage) LoadFile(todoID int, fileName string, w io.Writer) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	fileDir := filepath.Join(fs.dataDir, "files", fmt.Sprintf("todo_%d", todoID))
	filePath := filepath.Join(fileDir, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	br := bufio.NewReader(file)
	_, err = io.Copy(w, br)
	return err
}
