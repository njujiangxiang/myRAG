# myRAG Web 前端

基于 React + TypeScript + Vite 的 myRAG 知识库管理前端应用。

## 技术栈

- **React 19** - UI 框架
- **TypeScript** - 类型安全
- **Vite 5** - 构建工具
- **React Router** - 路由管理
- **shadcn/ui** - UI 组件库
- **Tailwind CSS 4** - 样式框架
- **Lucide React** - 图标库

## 功能特性

- ✅ 用户注册/登录
- ✅ 知识库管理（创建、列表、删除）
- ✅ 文档上传（支持 PDF、DOCX、MD、TXT、CSV 等格式）
- ✅ 文档状态实时查看
- ✅ 智能聊天界面（基于知识库内容的问答）
- ✅ 响应式设计（支持桌面和移动端）
- ✅ 深色模式支持

## 快速开始

### 开发模式

```bash
# 安装依赖
npm install

# 启动开发服务器（代理到后端 API）
npm run dev
```

访问 http://localhost:3000

### 生产构建

```bash
# 构建
npm run build

# 预览构建结果
npm run preview
```

### Docker 部署

```bash
# 构建并启动所有服务
docker compose up -d

# 查看日志
docker compose logs -f web
```

访问 http://localhost:3000

## 项目结构

```
web/
├── src/
│   ├── components/
│   │   ├── ui/          # shadcn/ui 组件
│   │   └── Layout.tsx   # 布局组件
│   ├── contexts/
│   │   └── AuthContext.tsx  # 认证上下文
│   ├── hooks/
│   │   └── useToast.ts  # Toast hook
│   ├── lib/
│   │   └── utils.ts     # 工具函数
│   ├── pages/
│   │   ├── LoginPage.tsx      # 登录页
│   │   ├── KBListPage.tsx     # 知识库列表
│   │   ├── KBDetailPage.tsx   # 知识库详情
│   │   └── ChatPage.tsx       # 聊天界面
│   ├── App.tsx        # 应用入口
│   ├── main.tsx       # React 入口
│   └── index.css      # 全局样式
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── Dockerfile
```

## API 代理

开发模式下，Vite 会代理 `/api` 请求到后端服务：

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
}
```

## 组件库

使用 shadcn/ui 组件库，所有组件位于 `src/components/ui/` 目录。

可用组件：
- Button
- Card
- Input
- Textarea
- DropdownMenu
- Progress
- Toast/Toaster

## 状态管理

- **认证状态**：`AuthContext` 管理用户登录状态和 JWT token
- **Toast 通知**：`useToast` hook 管理全局通知

## 路由

| 路径 | 组件 | 描述 |
|------|------|------|
| `/login` | LoginPage | 登录/注册页面 |
| `/kbs` | KBListPage | 知识库列表 |
| `/kbs/:id` | KBDetailPage | 知识库详情和文档管理 |
| `/kbs/:id/chat` | ChatPage | 智能聊天界面 |

## 开发注意事项

1. **TypeScript 严格模式**：所有代码必须通过 TypeScript 类型检查
2. **组件命名**：使用 PascalCase 命名组件
3. **样式**：使用 Tailwind CSS 类名，通过 `cn()` 工具函数合并类名
4. **图标**：使用 Lucide React 图标库

## 构建优化

- Gzip 压缩（Nginx）
- 静态资源缓存（1 年）
- 代码分割（自动）
- Tree shaking（自动）

## License

MIT
