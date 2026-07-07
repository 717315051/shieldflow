package main

import "golang.org/x/crypto/bcrypt"

// bcryptPassword 对密码进行 bcrypt 哈希
func bcryptPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
