package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alkmc/restClean/internal/cache"
	"github.com/alkmc/restClean/internal/entity"
	"github.com/alkmc/restClean/internal/repository"
	"github.com/alkmc/restClean/internal/service"
	"github.com/alkmc/restClean/internal/validator"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	NAME  = "Car"
	PRICE = 1.23
)

type responseMessage struct {
	Message string `json:"message"`
}

type mockRepo struct {
	repository.Repository
	save     func(p *entity.Product) (*entity.Product, error)
	findByID func(id uuid.UUID) (*entity.Product, error)
	findAll  func() ([]entity.Product, error)
	update   func(p *entity.Product) error
	delete   func(id uuid.UUID) error
}

func (m mockRepo) Save(ctx context.Context, p *entity.Product) (*entity.Product, error) {
	return m.save(p)
}
func (m mockRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.Product, error) {
	if m.findByID == nil {
		return nil, sql.ErrNoRows
	}
	return m.findByID(id)
}
func (m mockRepo) FindAll(ctx context.Context) ([]entity.Product, error) {
	return m.findAll()
}
func (m mockRepo) Update(ctx context.Context, p *entity.Product) error {
	return m.update(p)
}
func (m mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.delete(id)
}
func (m mockRepo) CloseDB() {}

func setupTest(t *testing.T) (http.Handler, *mockRepo) {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	repo := &mockRepo{}

	srv := service.NewService(repo)
	cacheSrv := cache.NewRedis(logger, "localhost:6379", 0, 10)
	valid := validator.NewValidator()
	h := NewHandler(logger, srv, cacheSrv, valid)
	mux := NewMux(logger, h)

	return mux, repo
}

func TestGetProductByID(t *testing.T) {
	mux, repo := setupTest(t)
	uid := uuid.New()

	tests := []struct {
		name           string
		id             string
		setupMock      func()
		expectedStatus int
		expectedMsg    string // for error cases
	}{
		{
			name: "success",
			id:   uid.String(),
			setupMock: func() {
				repo.findByID = func(id uuid.UUID) (*entity.Product, error) {
					return &entity.Product{ID: uid, Name: NAME, Price: PRICE}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "incorrect uuid",
			id:             "incorrect",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid UUID length: 9",
		},
		{
			name: "non-existing product",
			id:   uuid.New().String(),
			setupMock: func() {
				repo.findByID = func(id uuid.UUID) (*entity.Product, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "product not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			req := httptest.NewRequestWithContext(t.Context(), "GET", "/product/"+tt.id, nil)
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus == http.StatusOK {
				var p entity.Product
				err := json.NewDecoder(resp.Body).Decode(&p)
				require.NoError(t, err)
				assert.Equal(t, uid, p.ID)
				assert.Equal(t, NAME, p.Name)
			} else {
				var e responseMessage
				err := json.NewDecoder(resp.Body).Decode(&e)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMsg, e.Message)
			}
		})
	}
}

func TestGetProducts(t *testing.T) {
	mux, repo := setupTest(t)

	t.Run("empty", func(t *testing.T) {
		repo.findAll = func() ([]entity.Product, error) {
			return nil, nil
		}
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/product", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var e responseMessage
		err := json.NewDecoder(resp.Body).Decode(&e)
		require.NoError(t, err)
		assert.Equal(t, "no products found", e.Message)
	})

	t.Run("success", func(t *testing.T) {
		repo.findAll = func() ([]entity.Product, error) {
			return []entity.Product{{Name: NAME, Price: PRICE}}, nil
		}
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/product", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var products []entity.Product
		err := json.NewDecoder(resp.Body).Decode(&products)
		require.NoError(t, err)
		assert.NotEmpty(t, products)
		assert.Equal(t, NAME, products[0].Name)
	})
}

func TestAddProduct(t *testing.T) {
	mux, repo := setupTest(t)

	tests := []struct {
		name           string
		body           any
		setupMock      func()
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "success",
			body: entity.Product{Name: NAME, Price: PRICE},
			setupMock: func() {
				repo.save = func(p *entity.Product) (*entity.Product, error) {
					p.ID = uuid.New()
					return p, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "extra field",
			body:           map[string]any{"Name": NAME, "Price": PRICE, "Email": "a@a.com"},
			setupMock:      func() {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedMsg:    "unknown field \"Email\"",
		},
		{
			name:           "negative price",
			body:           entity.Product{Name: NAME, Price: -1.0},
			setupMock:      func() {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedMsg:    "the product price must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			b, err := json.Marshal(tt.body)
			require.NoError(t, err)

			req := httptest.NewRequestWithContext(t.Context(), "POST", "/product", bytes.NewBuffer(b))
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedMsg != "" {
				var e responseMessage
				err := json.NewDecoder(resp.Body).Decode(&e)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMsg, e.Message)
			}
		})
	}
}

func TestDeleteProduct(t *testing.T) {
	mux, repo := setupTest(t)
	uid := uuid.New()

	t.Run("not existing", func(t *testing.T) {
		repo.findByID = func(id uuid.UUID) (*entity.Product, error) {
			return nil, sql.ErrNoRows
		}
		req := httptest.NewRequestWithContext(t.Context(), "DELETE", "/product/"+uuid.New().String(), nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("success", func(t *testing.T) {
		repo.findByID = func(id uuid.UUID) (*entity.Product, error) {
			return &entity.Product{ID: uid, Name: NAME, Price: PRICE}, nil
		}
		repo.delete = func(id uuid.UUID) error {
			return nil
		}

		req := httptest.NewRequestWithContext(t.Context(), "DELETE", "/product/"+uid.String(), nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var e responseMessage
		err := json.NewDecoder(resp.Body).Decode(&e)
		require.NoError(t, err)
		assert.Equal(t, "product deleted", e.Message)
	})
}

func TestUpdateProduct(t *testing.T) {
	mux, repo := setupTest(t)
	uid := uuid.New()

	repo.findByID = func(id uuid.UUID) (*entity.Product, error) {
		return &entity.Product{ID: uid, Name: NAME, Price: PRICE}, nil
	}
	repo.update = func(p *entity.Product) error {
		return nil
	}

	update := entity.Product{ID: uid, Name: "Updated", Price: 99.9}
	b, err := json.Marshal(update)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), "PUT", "/product/"+uid.String(), bytes.NewBuffer(b))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var p entity.Product
	err = json.NewDecoder(resp.Body).Decode(&p)
	require.NoError(t, err)
	assert.Equal(t, "Updated", p.Name)
}
