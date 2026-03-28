package mongo

import (
	"context"
	"errors"

	usersService "nosql/internal/service/users"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	gomongo "go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	col *gomongo.Collection
}

func NewUserRepository(db *gomongo.Database) *UserRepository {
	return &UserRepository{
		col: db.Collection("users"),
	}
}

func (r *UserRepository) Create(ctx context.Context, user usersService.User) (string, error) {
	res, err := r.col.InsertOne(ctx, user)
	if err != nil {
		if gomongo.IsDuplicateKeyError(err) {
			return "", usersService.ErrUserExists
		}
		return "", err
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", errors.New("unexpected inserted id type")
	}

	return id.Hex(), nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (usersService.User, bool, error) {
	var user usersService.User

	err := r.col.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if errors.Is(err, gomongo.ErrNoDocuments) {
		return usersService.User{}, false, nil
	}
	if err != nil {
		return usersService.User{}, false, err
	}

	return user, true, nil
}
