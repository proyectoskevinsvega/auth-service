INSERT INTO auth_user_roles (tenant_id, user_id, role_id)
SELECT 
    t.id as tenant_id,
    u.id as user_id, 
    r.id as role_id
FROM auth_users u
JOIN auth_roles r ON 1=1
JOIN auth_tenants t ON u.tenant_id = t.id
WHERE u.email = 'yesidvegamar@gmail.com' AND r.name = 'admin';
