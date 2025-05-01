## 数据加密

在构建安全架构时，数据加密是保护数据不被未授权访问的第一道防线。通过加密，即使数据在传输或存储过程中被截获，其内容也不会轻易泄露。常见的加密算法包括AES（高级加密标准）和RSA（公钥加密算法）。

### 示例：使用Python进行AES加密

```python
from Crypto.Cipher import AES
from Crypto.Random import get_random_bytes

# 生成密钥
key = get_random_bytes(16)

# 创建AES加密器
cipher = AES.new(key, AES.MODE_ECB)

# 要加密的数据
data = b"Hello, World!"

# 数据需要是16字节的倍数
data += b' ' * (16 - len(data) % 16)

# 加密数据
encrypted_data = cipher.encrypt(data)
print("Encrypted:", encrypted_data)
```

## 访问控制

访问控制是确保只有授权用户才能访问特定资源的关键机制。通过实施访问控制策略，可以有效防止未授权的访问和操作，常见的访问控制模型包括RBAC（基于角色的访问控制）和ABAC（基于属性的访问控制）。

### 示例：使用Python实现简单的RBAC

```python
class User:
    def __init__(self, roles):
        self.roles = roles

class Resource:
    def __init__(self, required_role):
        self.required_role = required_role

    def access(self, user):
        if self.required_role in user.roles:
            print("Access granted!")
        else:
            print("Access denied!")

# 创建用户和角色
user = User(roles=['admin', 'user'])
resource = Resource(required_role='admin')

# 尝试访问资源
resource.access(user)
```

## 身份认证

身份认证是验证用户身份的过程，确保系统仅响应合法用户。常见的身份认证方法包括密码认证、双因素认证和生物特征认证。使用安全的身份认证机制可以大大减少被冒充的风险。

### 示例：使用Python实现简单的密码认证

```python
from getpass import getpass

def authenticate(username, password):
    # 假设这是存储的用户信息
    users = {
        'alice': 'password123',
        'bob': 'secure456'
    }
    
    if username in users and users[username] == password:
        print("Authentication successful!")
    else:
        print("Authentication failed!")

# 获取用户输入
username = input("Enter your username: ")
password = getpass("Enter your password: ")

# 调用认证函数
authenticate(username, password)
```

通过上述示例，我们可以看到数据加密、访问控制和身份认证在构建安全架构中的重要性。这些技术的正确实施可以显著提高系统的安全性。