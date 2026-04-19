package mongo

import (
	"context"
	"errors"
	"regexp"

	usersService "nosql/internal/service/users"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	gomongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (r *UserRepository) FindByID(ctx context.Context, id string) (usersService.User, bool, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return usersService.User{}, false, nil
	}

	var user usersService.User
	err = r.col.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if errors.Is(err, gomongo.ErrNoDocuments) {
		return usersService.User{}, false, nil
	}
	if err != nil {
		return usersService.User{}, false, err
	}

	return user, true, nil
}

func (r *UserRepository) List(ctx context.Context, filter usersService.ListFilter) ([]usersService.User, error) {
	mongoFilter := bson.M{}

	if filter.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(filter.ID)
		if err != nil {
			return []usersService.User{}, nil
		}
		mongoFilter["_id"] = objectID
	}
	if filter.Name != "" {
		mongoFilter["full_name"] = bson.M{
			"$regex": regexp.QuoteMeta(filter.Name),
		}
	}

	opts := options.Find()
	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}

	cursor, err := r.col.Find(ctx, mongoFilter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []usersService.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}
