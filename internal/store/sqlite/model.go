package sqlite

import "time"

type User struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	IsSuper      bool      `gorm:"not null;default:false" json:"is_super"`
	RoleID       *string   `gorm:"size:36" json:"role_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

type Role struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Role) TableName() string {
	return "roles"
}

type RolePermission struct {
	ID       string `gorm:"primaryKey;size:36" json:"id"`
	RoleID   string `gorm:"size:36;not null;index:idx_role_module,unique" json:"role_id"`
	Module   string `gorm:"size:32;not null;index:idx_role_module,unique" json:"module"`
	CanRead  bool   `gorm:"not null;default:false" json:"can_read"`
	CanWrite bool   `gorm:"not null;default:false" json:"can_write"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

type RoleEnvironment struct {
	ID            string `gorm:"primaryKey;size:36" json:"id"`
	RoleID        string `gorm:"size:36;not null;index:idx_role_env,unique" json:"role_id"`
	EnvironmentID string `gorm:"size:36;not null;index:idx_role_env,unique" json:"environment_id"`
}

func (RoleEnvironment) TableName() string {
	return "role_environments"
}

type Environment struct {
	ID            string    `gorm:"primaryKey;size:36" json:"id"`
	Name          string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	KeyPrefix     string    `gorm:"size:255;not null" json:"key_prefix"`
	ConfigPrefix  string    `gorm:"size:255;not null;default:config/" json:"config_prefix"`
	GatewayPrefix string    `gorm:"size:255;not null;default:gw-services/" json:"gateway_prefix"`
	GrpcPrefix    string    `gorm:"size:255;not null;default:grpc-services/" json:"grpc_prefix"`
	Description   string    `gorm:"size:255" json:"description"`
	SortOrder     int       `gorm:"default:0" json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Environment) TableName() string {
	return "environments"
}

type ConfigRevision struct {
	ID            string    `gorm:"primaryKey;size:36" json:"id"`
	EnvironmentID string    `gorm:"size:36;index;not null" json:"environment_id"`
	Key           string    `gorm:"size:512;not null;index" json:"key"`
	Value         string    `gorm:"type:text" json:"value"`
	PrevValue     string    `gorm:"type:text" json:"prev_value"`
	EtcdRevision  int64     `json:"etcd_revision"`
	Action        string    `gorm:"size:16;not null" json:"action"`
	Operator      string    `gorm:"size:36" json:"operator"`
	Comment       string    `gorm:"size:512" json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ConfigRevision) TableName() string {
	return "config_revisions"
}

type AuditLog struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	UserID       string    `gorm:"size:36;index" json:"user_id"`
	Action       string    `gorm:"size:64;not null;index" json:"action"`
	ResourceType string    `gorm:"size:64;not null" json:"resource_type"`
	ResourceKey  string    `gorm:"size:512" json:"resource_key"`
	Detail       string    `gorm:"type:text" json:"detail"`
	IP           string    `gorm:"size:45" json:"ip"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
