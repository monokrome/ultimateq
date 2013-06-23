package data

import (
	. "testing"
)

func TestStore(t *T) {
	t.Parallel()
	s, err := CreateStore(MemStoreProvider)
	if err != nil {
		t.Fatal(err)
	}

	if s.cache == nil {
		t.Error("Cache not instantiated.")
	}

	err = s.Close()
	if err != nil {
		t.Error("Closing database failed.")
	}
}

func TestStore_AddUser(t *T) {
	t.Parallel()
	s, err := CreateStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(s.cache) > 0 {
		t.Error("Pre-warmed cache somehow exists.")
	}

	ua1 := &UserAccess{Username: uname}
	ua2 := &UserAccess{Username: uname + uname}

	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	if s.cache[ua1.Username] == nil {
		t.Error("User was not cached.")
	}

	err = s.AddUser(ua2)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	if s.cache[ua1.Username] != nil {
		t.Error("User should no longer be cached due to caching limits.")
	}
	if s.cache[ua2.Username] == nil {
		t.Error("User was not cached.")
	}

	found, err := s.fetchUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("The user was not found.")
	}
	found, err = s.fetchUser(ua2.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("The user was not found.")
	}
}

func TestStore_RemoveUser(t *T) {
	t.Parallel()
	s, err := CreateStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	ua1 := &UserAccess{Username: uname}

	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	if s.cache[ua1.Username] == nil {
		t.Error("User was not cached.")
	}

	found, err := s.fetchUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("Error fetching user.")
	}

	err = s.RemoveUser(ua1.Username)
	if err != nil {
		t.Fatal("Error removing user:", err)
	}
	if s.cache[ua1.Username] != nil {
		t.Error("User is still cached.")
	}

	found, err = s.fetchUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found != nil {
		t.Error("User should be removed.")
	}
}

func TestStore_AuthUser(t *T) {
	t.Parallel()
	s, err := CreateStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	ua1, err := CreateUserAccess(uname, password, `*!*@host`)
	if err != nil {
		t.Fatal("Error creating user:", err)
	}
	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	user, err := s.AuthUser(server, host, uname+uname, password)
	if user != nil {
		t.Error("Failed to reject bad authentication.")
	}
	if err != ErrUserNotFound {
		t.Errorf("Expected error %v but got %v", ErrUserNotFound, err)
	}

	user, err = s.AuthUser(server, `nick!user@host.com`, uname, password)
	if user != nil {
		t.Error("Failed to reject bad authentication.")
	}
	if err != ErrUserBadHost {
		t.Errorf("Expected error %v but got %v", ErrUserBadHost, err)
	}

	user, err = s.AuthUser(server, host, uname, password+password)
	if user != nil {
		t.Error("Failed to reject bad authentication.")
	}
	if err != ErrUserBadPassword {
		t.Errorf("Expected error %v but got %v", ErrUserBadPassword, err)
	}

	user, err = s.AuthUser(server, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}

	if s.authed[server+host] == nil {
		t.Error("User is not authenticated.")
	}

	// Testing previously authenticated look up.
	user, err = s.AuthUser(server, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}
}

func TestStore_AuthLogout(t *T) {
	t.Parallel()
	s, err := CreateStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	ua1, err := CreateUserAccess(uname, password)
	if err != nil {
		t.Fatal("Error creating user:", err)
	}
	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	s.cache = make(map[string]*UserAccess)

	user, err := s.AuthUser(server, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}

	if len(s.cache) == 0 {
		t.Error("Auth is not using cache.")
	}
	if s.authed[server+host] == nil {
		t.Error("User is not authenticated.")
	}

	s.Logout(server, host)

	if s.authed[server+host] != nil {
		t.Error("User is still authenticated.")
	}
}

func TestStore_Finding(t *T) {
	t.Parallel()
	s, err := CreateStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(s.cache) > 0 {
		t.Error("Pre-warmed cache somehow exists.")
	}

	ua1 := &UserAccess{Username: uname}
	ua2 := &UserAccess{Username: uname + uname}

	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Could not add user.")
	}
	err = s.AddUser(ua2)
	if err != nil {
		t.Fatal("Could not add user.")
	}

	s.cache = make(map[string]*UserAccess)

	found, err := s.fetchUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}
	found, err = s.fetchUser(ua2.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}

	if len(s.cache) > 0 {
		t.Error("Cache should not be warmed by fetchUser.")
	}

	found, err = s.findUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}
	// Cached lookup, for test coverage.
	found, err = s.findUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}
	found, err = s.findUser(ua2.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}

	if len(s.cache) != nMaxCache {
		t.Error("Cache should be being used.")
	}
}
