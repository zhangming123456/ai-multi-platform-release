from fastapi.testclient import TestClient
from app.main import app


def main():
    with TestClient(app) as client:
        # 1. Admin login
        r = client.post('/api/auth/login', json={'username': 'admin', 'password': 'admin123'})
        print('admin login', r.status_code)
        if r.status_code != 200:
            print(r.text)
            raise SystemExit(1)
        admin_token = r.json()['access_token']
        admin_headers = {'Authorization': f'Bearer {admin_token}'}

        # auth me
        r = client.get('/api/auth/me', headers=admin_headers)
        print('admin me', r.status_code)
        admin_perms = r.json().get('permissions', {})
        rbac = {k: v for k, v in admin_perms.items() if k.startswith(('users:', 'roles:', 'permissions:', 'constraints:'))}
        print('admin rbac perms', rbac)

        # 2. Operator login
        r = client.post('/api/auth/login', json={'username': 'operator', 'password': 'operator123'})
        print('operator login', r.status_code)
        if r.status_code != 200:
            print(r.text)
            raise SystemExit(1)
        op_token = r.json()['access_token']
        op_headers = {'Authorization': f'Bearer {op_token}'}
        r = client.get('/api/auth/me', headers=op_headers)
        op_perms = r.json().get('permissions', {})
        op_rbac = {k: v for k, v in op_perms.items() if k.startswith(('users:', 'roles:', 'permissions:', 'constraints:'))}
        print('operator rbac perms', op_rbac)

        # 3. Create parent role and assign permission
        r = client.post('/api/v2/roles', json={
            'name': 'test_parent',
            'display_name': '测试父角色',
            'description': '',
            'role_type': 'other',
            'parent_role_ids': []
        }, headers=admin_headers)
        print('create parent role', r.status_code)
        parent_role = r.json()

        # Get permissions
        r = client.get('/api/v2/permissions', headers=admin_headers)
        perms = r.json()
        content_read = next((p for p in perms if p['key'] == 'content:read'), None)
        content_create = next((p for p in perms if p['key'] == 'content:create'), None)
        print('content:read', content_read is not None, 'content:create', content_create is not None)

        # Assign content:read to parent
        if content_read:
            url = '/api/v2/roles/' + parent_role['id'] + '/permissions'
            r = client.put(url, json={'permission_ids': [content_read['id']]}, headers=admin_headers)
            print('assign parent perm', r.status_code)

        # Create child role with parent
        r = client.post('/api/v2/roles', json={
            'name': 'test_child',
            'display_name': '测试子角色',
            'description': '',
            'role_type': 'other',
            'parent_role_ids': [parent_role['id']]
        }, headers=admin_headers)
        print('create child role', r.status_code)
        child_role = r.json()

        # Check child inherited permissions
        url = '/api/v2/roles/' + child_role['id'] + '/permissions/detail'
        r = client.get(url, headers=admin_headers)
        print('child detail', r.status_code, r.json())

        # Assign custom permission to child
        if content_create:
            url = '/api/v2/roles/' + child_role['id'] + '/permissions'
            r = client.put(url, json={'permission_ids': [content_create['id']]}, headers=admin_headers)
            print('assign child custom perm', r.status_code)
            url = '/api/v2/roles/' + child_role['id'] + '/permissions/detail'
            r = client.get(url, headers=admin_headers)
            print('child detail after custom', r.status_code, r.json())

        # 4. Mutual exclusion constraint
        r = client.post('/api/v2/roles', json={
            'name': 'mutex_a',
            'display_name': '互斥A',
            'role_type': 'other'
        }, headers=admin_headers)
        role_a = r.json()
        r = client.post('/api/v2/roles', json={
            'name': 'mutex_b',
            'display_name': '互斥B',
            'role_type': 'other'
        }, headers=admin_headers)
        role_b = r.json()

        r = client.post('/api/v2/constraints', json={
            'name': 'test_mutex',
            'constraint_type': 'mutual_exclusive',
            'config': {'scope': 'static'},
            'role_associations': [
                {'role_id': role_a['id'], 'association_type': 'subject'},
                {'role_id': role_b['id'], 'association_type': 'subject'}
            ]
        }, headers=admin_headers)
        print('create mutex constraint', r.status_code, r.text)

        # Try assign both to operator
        r = client.get('/api/v2/users', headers=admin_headers)
        users = r.json()
        op_user = next((u for u in users if u['username'] == 'operator'), None)
        if op_user:
            url = '/api/v2/users/' + op_user['id'] + '/roles'
            r = client.put(url, json={'role_ids': [role_a['id'], role_b['id']]}, headers=admin_headers)
            print('assign mutex roles', r.status_code, r.text)


if __name__ == '__main__':
    main()
