# Proof of Reserve (PoR) - Monorepo

This project provides a full-stack Proof of Reserve system with a smart contract, backend, and frontend.

## Prerequisites
- [Docker](https://www.docker.com/get-started)
- [Docker Compose](https://docs.docker.com/compose/)

## Quick Start (Docker Compose)

From the `packages` directory:

```sh
cd packages
cp por-frontend/.env.example por-frontend/.env # Edit as needed
cp backend/.env.example backend/.env           # Edit as needed
# Build and start all services
docker compose up --build
```

- Frontend: http://localhost:5173
- Backend:  http://localhost:8082

## Manual Build & Run (for development)

### Backend
```sh
cd backend
cp .env.example .env # Edit as needed
go run cmd/server/main.go
```

### Frontend
```sh
cd por-frontend
cp .env.example .env # Edit as needed
npm install
npm run dev
```

## Environment Variables
See `.env.example` files in each package for required configuration (API URLs, contract address, etc).

## Contracts
See `../contracts/` for Solidity source and deployment instructions.

---

## Docker Compose Services
- **backend**: Go Gin API server
- **por-frontend**: Vite/React frontend

---

## Troubleshooting
- Ensure your `.env` files are set up and correct.
- Check logs with `docker compose logs -f`.
- For contract issues, check your RPC URL and contract address.

# React + TypeScript + Vite

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Babel](https://babeljs.io/) for Fast Refresh
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/) for Fast Refresh

## Expanding the ESLint configuration

If you are developing a production application, we recommend updating the configuration to enable type-aware lint rules:

```js
export default tseslint.config({
  extends: [
    // Remove ...tseslint.configs.recommended and replace with this
    ...tseslint.configs.recommendedTypeChecked,
    // Alternatively, use this for stricter rules
    ...tseslint.configs.strictTypeChecked,
    // Optionally, add this for stylistic rules
    ...tseslint.configs.stylisticTypeChecked,
  ],
  languageOptions: {
    // other options...
    parserOptions: {
      project: ['./tsconfig.node.json', './tsconfig.app.json'],
      tsconfigRootDir: import.meta.dirname,
    },
  },
})
```

You can also install [eslint-plugin-react-x](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-x) and [eslint-plugin-react-dom](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-dom) for React-specific lint rules:

```js
// eslint.config.js
import reactX from 'eslint-plugin-react-x'
import reactDom from 'eslint-plugin-react-dom'

export default tseslint.config({
  plugins: {
    // Add the react-x and react-dom plugins
    'react-x': reactX,
    'react-dom': reactDom,
  },
  rules: {
    // other rules...
    // Enable its recommended typescript rules
    ...reactX.configs['recommended-typescript'].rules,
    ...reactDom.configs.recommended.rules,
  },
})
```
