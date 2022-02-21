package mongo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/modernice/goes/persistence/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ model.Repository[model.Model[uuid.UUID], uuid.UUID] = &ModelRepository[model.Model[uuid.UUID], uuid.UUID]{}

// ModelRepository is a MongoDB backed model repository.
type ModelRepository[Model model.Model[ID], ID model.ID] struct {
	modelRepositoryOptions
	col *mongo.Collection
}

// ModelRepositoryOption is an option for the model repository.
type ModelRepositoryOption func(*modelRepositoryOptions)

type modelRepositoryOptions struct {
	key          string
	transactions bool
}

// ModelIDKey returns a ModelRepositoryOption that specifies which field of the
// model is the id of the model.
func ModelIDKey(key string) ModelRepositoryOption {
	return func(o *modelRepositoryOptions) {
		o.key = key
	}
}

// ModelTransactions returns a ModelRepositoryOption that enables MongoDB
// transactions for the repository. Currently, only the Use() function makes use
// of transactions. Transactions are disabled by default and must be supported
// by your MongoDB cluster.
func ModelTransactions(tx bool) ModelRepositoryOption {
	return func(o *modelRepositoryOptions) {
		o.transactions = tx
	}
}

// NewModelRepository returns a MongoDB backed model repository.
func NewModelRepository[Model model.Model[ID], ID model.ID](col *mongo.Collection, opts ...ModelRepositoryOption) *ModelRepository[Model, ID] {
	var options modelRepositoryOptions
	for _, opt := range opts {
		opt(&options)
	}

	if options.key == "" {
		options.key = "_id"
	}

	return &ModelRepository[Model, ID]{
		modelRepositoryOptions: options,
		col:                    col,
	}
}

// Save saves the given model to the database using the MongoDB "ReplaceOne"
// command with the upsert option set to true.
func (r *ModelRepository[Model, ID]) Save(ctx context.Context, m Model) error {
	_, err := r.col.ReplaceOne(ctx, bson.D{{Key: r.key, Value: m.ModelID()}}, m, options.Replace().SetUpsert(true))
	return err
}

// Fetch fetches the given model from the database. If the model cannot be found,
// an error that unwraps to model.ErrNotFound is returned.
func (r *ModelRepository[Model, ID]) Fetch(ctx context.Context, id ID) (Model, error) {
	res := r.col.FindOne(ctx, bson.D{{Key: r.key, Value: id}})

	var m Model
	if err := res.Decode(&m); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return m, fmt.Errorf("%w: %v", model.ErrNotFound, err)
		}
		return m, fmt.Errorf("decode model: %w", err)
	}

	return m, nil
}

// Use fetches the given model from the database, passes the model to the
// provided function and finally saves the model back to the database.
// If the ModelTransactions option is set to true, the operation is done within
// a MongoDB transaction (must be supported by your MongoDB cluster).
func (r *ModelRepository[Model, ID]) Use(ctx context.Context, id ID, fn func(Model) error) error {
	return r.col.Database().Client().UseSession(ctx, func(ctx mongo.SessionContext) error {
		if err := ctx.StartTransaction(); err != nil {
			return fmt.Errorf("start transaction: %w", err)
		}

		m, err := r.Fetch(ctx, id)
		if err != nil {
			return fmt.Errorf("fetch model: %w", err)
		}

		if err := fn(m); err != nil {
			return err
		}

		if err := r.Save(ctx, m); err != nil {
			return fmt.Errorf("save model: %w", err)
		}

		if err := ctx.CommitTransaction(ctx); err != nil {
			return fmt.Errorf("commit transaction: %w", err)
		}

		return nil
	})
}

// Delete deletes the given model from the database.
func (r *ModelRepository[Model, ID]) Delete(ctx context.Context, m Model) error {
	_, err := r.col.DeleteOne(ctx, bson.D{{Key: r.key, Value: m.ModelID()}})
	return err
}