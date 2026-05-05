package service

import (
	"context"
	"testing"

	"github.com/alkmc/restClean/internal/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockRepository struct {
	SaveFn     func(ctx context.Context, p *entity.Product) (*entity.Product, error)
	FindByIDFn func(ctx context.Context, id uuid.UUID) (*entity.Product, error)
	FindAllFn  func(ctx context.Context) ([]entity.Product, error)
	UpdateFn   func(ctx context.Context, p *entity.Product) error
	DeleteFn   func(ctx context.Context, id uuid.UUID) error
	CloseDBFn  func()
}

func (m *MockRepository) Save(ctx context.Context, p *entity.Product) (*entity.Product, error) {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, p)
	}
	return nil, nil
}

func (m *MockRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Product, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockRepository) FindAll(ctx context.Context) ([]entity.Product, error) {
	if m.FindAllFn != nil {
		return m.FindAllFn(ctx)
	}
	return nil, nil
}

func (m *MockRepository) Update(ctx context.Context, p *entity.Product) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, p)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) CloseDB() {
	if m.CloseDBFn != nil {
		m.CloseDBFn()
	}
}

func TestProductService(t *testing.T) {
	ctx := t.Context()

	t.Run("Create", func(t *testing.T) {
		p := &entity.Product{Name: "Test", Price: 10.0}
		mockRepo := &MockRepository{
			SaveFn: func(ctx context.Context, p *entity.Product) (*entity.Product, error) {
				return p, nil
			},
		}
		srv := NewService(mockRepo)

		res, err := srv.Create(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, "Test", res.Name)
	})

	t.Run("FindByID", func(t *testing.T) {
		id := uuid.New()
		p := &entity.Product{ID: id, Name: "Test"}
		mockRepo := &MockRepository{
			FindByIDFn: func(ctx context.Context, id uuid.UUID) (*entity.Product, error) {
				return p, nil
			},
		}
		srv := NewService(mockRepo)

		res, err := srv.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, id, res.ID)
	})

	t.Run("FindAll", func(t *testing.T) {
		products := []entity.Product{{Name: "P1"}, {Name: "P2"}}
		mockRepo := &MockRepository{
			FindAllFn: func(ctx context.Context) ([]entity.Product, error) {
				return products, nil
			},
		}
		srv := NewService(mockRepo)

		res, err := srv.FindAll(ctx)
		require.NoError(t, err)
		assert.Len(t, res, 2)
	})

	t.Run("Update", func(t *testing.T) {
		p := &entity.Product{Name: "Update"}
		mockRepo := &MockRepository{
			UpdateFn: func(ctx context.Context, p *entity.Product) error {
				return nil
			},
		}
		srv := NewService(mockRepo)

		err := srv.Update(ctx, p)
		require.NoError(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		id := uuid.New()
		mockRepo := &MockRepository{
			DeleteFn: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}
		srv := NewService(mockRepo)

		err := srv.Delete(ctx, id)
		require.NoError(t, err)
	})
}
