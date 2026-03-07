package user

import (
	"log/slog"
	"maps"
	"slices"
	"sync"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
)

type Registry struct {
	mu     sync.RWMutex
	users  map[uuid.UUID]*model.User
	logger *slog.Logger
}

func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		users:  make(map[uuid.UUID]*model.User),
		logger: logger,
	}
}

func (r *Registry) CreateUser(firstName, lastName, name string, additionalInfo model.AdditionalInfo) *model.User {
	user := &model.User{
		ID:             uuid.New(),
		FirstName:      firstName,
		LastName:       lastName,
		Name:           name,
		AdditionalInfo: additionalInfo,
	}

	r.mu.Lock()
	r.users[user.ID] = user
	r.mu.Unlock()

	r.logger.Info("user created", "userID", user.ID, "firstName", firstName, "lastName", lastName, "name", name)
	return user
}

func (r *Registry) GetUser(id uuid.UUID) (*model.User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[id]
	return user, ok
}

func (r *Registry) GetAllUsers() []*model.User {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return slices.Collect(maps.Values(r.users))
}

func (r *Registry) UpdateUser(id uuid.UUID, firstName, lastName, name string, additionalInfo model.AdditionalInfo) (*model.User, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[id]
	if !ok {
		return nil, false
	}

	user.FirstName = firstName
	user.LastName = lastName
	user.Name = name
	user.AdditionalInfo = additionalInfo

	r.logger.Info("user updated", "userID", id)
	return user, true
}

func (r *Registry) PatchUser(id uuid.UUID, updates map[string]any) (*model.User, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[id]
	if !ok {
		return nil, false
	}

	if firstName, ok := updates["firstName"].(string); ok {
		user.FirstName = firstName
	}
	if lastName, ok := updates["lastName"].(string); ok {
		user.LastName = lastName
	}
	if name, ok := updates["name"].(string); ok {
		user.Name = name
	}
	if additionalInfo, ok := updates["additionalInfo"].(map[string]any); ok {
		if user.AdditionalInfo == nil {
			user.AdditionalInfo = make(model.AdditionalInfo)
		}
		maps.Copy(user.AdditionalInfo, additionalInfo)
	}

	r.logger.Info("user patched", "userID", id)
	return user, true
}

func (r *Registry) DeleteUser(id uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[id]; !ok {
		return false
	}

	delete(r.users, id)
	r.logger.Info("user deleted", "userID", id)
	return true
}
