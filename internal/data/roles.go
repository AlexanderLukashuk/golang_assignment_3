package data

// type Role struct {
// 	ID       int64  `json:"id"`
// 	RoleName string `json:"role_name"`
// 	UserID   int64  `json:"user_id"`
// }

// // Define a MovieModel struct type which wraps a sql.DB connection pool.
// type RoleModel struct {
// 	DB *sql.DB
// }

func (m RoleModel) Insert(role *Role) error {
	query := `
		INSERT INTO roles(role_name, user_id)
		VALUES ($1, $2)
		RETURNING id, created_at, version`

	return m.DB.QueryRow(query, &role.RoleName, &role.UserID).Scan(&role.ID, &role.RoleName, &role.UserID)
}

func Create(roleID int64, roleName string, userID int64) (*Role, error) {
	role := &Role{
		ID:       roleID,
		RoleName: roleName,
		UserID:   userID,
	}

	return role, nil
}

func (m RoleModel) NewRole(roleID int64, roleName string, userID int64) (*Role, error) {
	role, err := Create(roleID, roleName, userID)
	if err != nil {
		return nil, err
	}
	err = m.Insert(role)
	return role, err
}
