package seed

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/dysodeng/config-center/internal/domain"
)

func CreateAdminUser(ctx context.Context, userRepo domain.UserRepository) error {
	if _, err := userRepo.GetByUsername(ctx, "admin"); err == nil {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := userRepo.Create(ctx, &domain.User{
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         "admin",
	}); err != nil {
		return err
	}
	fmt.Println("========================================")
	fmt.Println("  Default admin user created:")
	fmt.Println("  Username: admin")
	fmt.Println("  Password: admin123")
	fmt.Println("  Please change the password after login!")
	fmt.Println("========================================")
	return nil
}
