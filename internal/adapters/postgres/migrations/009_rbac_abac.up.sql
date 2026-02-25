-- Migration v9: RBAC & ABAC

-- 1. Create Roles table
CREATE TABLE IF NOT EXISTS auth_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Create Permissions table
CREATE TABLE IF NOT EXISTS auth_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Create Role-Permissions joining table
CREATE TABLE IF NOT EXISTS auth_role_permissions (
    role_id UUID REFERENCES auth_roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES auth_permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- 4. Create User-Roles joining table
CREATE TABLE IF NOT EXISTS auth_user_roles (
    user_id UUID REFERENCES auth_users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES auth_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- 5. Add attributes column for ABAC
ALTER TABLE auth_users ADD COLUMN IF NOT EXISTS attributes JSONB DEFAULT '{}';

-- 6. Insert default roles
INSERT INTO auth_roles (name, description) VALUES 
('admin', 'Super administrator with full access'),
('user', 'Basic user with limited access'),
('guest', 'Guest user with read-only access')
ON CONFLICT (name) DO NOTHING;

-- 7. Insert basic permissions
INSERT INTO auth_permissions (name, description) VALUES 
('users:read', 'Ability to view user profiles'),
('users:write', 'Ability to create or update user profiles'),
('users:delete', 'Ability to delete users'),
('roles:read', 'Ability to view roles and permissions'),
('roles:manage', 'Ability to create, update or delete roles'),
('auth:force_reset', 'Ability to force password reset on users')
ON CONFLICT (name) DO NOTHING;

-- 8. Assign all permissions to admin
INSERT INTO auth_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM auth_roles r, auth_permissions p 
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- 9. Assign read permission to user
INSERT INTO auth_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM auth_roles r, auth_permissions p 
WHERE r.name = 'user' AND p.name = 'users:read'
ON CONFLICT DO NOTHING;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_auth_users_attributes ON auth_users USING GIN (attributes);
