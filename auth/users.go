package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type User struct {
	// user credentails
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UserName  string    `json:"user_name"`
	Password  string    `json:"-"`
	Email     string    `json:"email"`
	LastLogin time.Time `json:"last_login"`

	// used for maintenance roles
	Admin bool `json:"admin"`

	// sfs/users/this user
	SvcRoot string `json:"svc_root"`

	// path to user state file, ie:
	// sfs/users/state/state.json
	SfPath string `json:"state_file"`

	// drive info
	DriveID    string `json:"drive_id"`
	TotalFiles int    `json:"total_files"`
	TotalDirs  int    `json:"total_dirs"`

	// path to the the drive root for their filesystem, ie:
	// sfs/users/user-who-ever/root
	DrvRoot string `json:"root"`
}

func valid(name, userName, email, svcRoot string) bool {
	if name == "" || userName == "" || email == "" || svcRoot == "" {
		return false
	}
	return true
}

func NewUser(name string, userName string, email string, svcRoot string, isAdmin bool) *User {
	if !valid(name, userName, email, svcRoot) {
		log.Fatalf("all new user params must be provided")
	}
	return &User{
		ID:        NewUUID(),
		Name:      name,
		UserName:  userName,
		Password:  "default",
		Email:     email,
		LastLogin: time.Now().UTC(),
		Admin:     isAdmin,
		SvcRoot:   svcRoot,
		SfPath:    "", // set the first time the state is saved
		DriveID:   "", // set during first time set up
		DrvRoot:   "", // set during first time set up
	}
}

// convert the curent user state to a json-formatted byte slice
func (u *User) ToJSON() ([]byte, error) {
	data, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
}

func UnmarshalUserStr(userInfo string) (*User, error) {
	newUser := new(User)
	if err := json.Unmarshal([]byte(userInfo), &newUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user data: %v", err)
	}
	return newUser, nil
}
