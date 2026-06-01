# AI-DESK Frontend Setup Guide

## Project Overview

Professional React 19 + TypeScript ticket management system with clean, minimalist design. Built with Tailwind CSS v4 and shadcn/ui components.

## Quick Start

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

The app will run on `http://localhost:5173` by default.

## Project Structure

```
frontend/
├── src/
│   ├── components/           # Reusable components
│   │   ├── ui/              # shadcn/ui components (Button, Input, Select)
│   │   ├── Header.tsx        # Top navigation bar
│   │   ├── Sidebar.tsx       # Left sidebar navigation
│   │   ├── MainLayout.tsx    # Main layout wrapper
│   │   └── ProtectedRoute.tsx # Auth guard for routes
│   ├── pages/               # Page components
│   │   ├── LoginPage.tsx     # Login form
│   │   ├── DashboardPage.tsx # Ticket list with filters
│   │   ├── TicketDetailPage.tsx # Single ticket detail view
│   │   ├── CustomersPage.tsx # Customer management
│   │   ├── EngineersPage.tsx # Engineer management
│   │   └── ReportsPage.tsx   # Reports/analytics
│   ├── hooks/               # Custom React hooks
│   │   ├── useAuth.ts       # Authentication logic
│   │   └── useTickets.ts    # Ticket data fetching
│   ├── services/            # API integration
│   │   └── api.ts          # Axios instance with interceptors
│   ├── types/              # TypeScript types
│   │   └── index.ts        # All type definitions
│   ├── styles/             # Global styles
│   │   └── globals.css     # Tailwind + custom utilities
│   ├── lib/                # Utility functions
│   │   └── utils.ts        # Helper functions
│   ├── App.tsx             # Main app component with routing
│   └── main.tsx            # React entry point
├── index.html              # HTML template
├── package.json            # Dependencies
├── tsconfig.json           # TypeScript config
├── vite.config.ts          # Vite config
├── tailwind.config.js      # Tailwind theme config
├── postcss.config.js       # PostCSS config
├── .env.example            # Environment variables template
└── .gitignore              # Git ignore rules
```

## Key Features

### Authentication
- JWT token-based authentication
- Protected routes with automatic redirect to login
- Token stored in localStorage
- Auto-logout on 401 response

### Dashboard
- Sortable ticket table with pagination
- Real-time filters: Status, Priority, Customer, Search
- Responsive design (desktop table, mobile cards)
- Quick actions to view ticket details

### Ticket Details
- Full ticket information display
- Comment section with add/view comments
- Activity timeline showing all updates
- Status change dropdown
- Related customer & engineer info

### Customer Management
- View all customers
- Add new customers
- Edit customer details
- Delete customers
- Pagination support

### Engineer Management
- View support team members
- Add new engineers
- Edit engineer profiles
- Delete engineers
- Status management (active/inactive)

### Design System
- Neutral gray color palette with blue accent (#2E75B6)
- Consistent spacing and typography
- Shadow system for depth
- Accessibility-first approach
- Mobile-first responsive design

## Environment Variables

Create `.env.local` from `.env.example`:

```
VITE_API_BASE_URL=http://localhost:8000/api
VITE_APP_NAME=AI-DESK
```

## API Integration

All API calls go through `src/services/api.ts`:

- Automatic JWT token injection
- Error handling with 401 logout
- Request/response interceptors
- Type-safe endpoints
- Base URL configuration via env vars

## Tech Stack

- **React 19**: Latest UI framework
- **TypeScript**: Type safety and better DX
- **Tailwind CSS v4**: Utility-first styling
- **React Router v6**: Client-side routing
- **Axios**: HTTP client
- **Lucide React**: Icon library
- **date-fns**: Date formatting
- **Vite**: Lightning-fast build tool

## Available Scripts

```bash
# Development server
npm run dev

# Type checking
npm run type-check

# Production build
npm run build

# Preview production build locally
npm run preview

# Linting
npm run lint
```

## Component Usage

### Button Component
```tsx
<Button variant="default">Primary</Button>
<Button variant="outline">Secondary</Button>
<Button variant="ghost">Tertiary</Button>
<Button variant="destructive">Danger</Button>
<Button size="sm">Small</Button>
<Button size="lg">Large</Button>
<Button size="icon"><Icon /></Button>
```

### Protected Routes
```tsx
<ProtectedRoute>
  <DashboardPage />
</ProtectedRoute>
```

### Custom Hooks
```tsx
const { user, isAuthenticated, login, logout } = useAuth();
const { tickets, pagination, fetchTickets } = useTickets();
```

## Design Principles

1. **Minimalist**: No unnecessary animations or clutter
2. **Professional**: Clean typography and color palette
3. **Functional**: Focus on usability over decoration
4. **Responsive**: Works perfectly on all devices
5. **Accessible**: Semantic HTML and proper ARIA labels
6. **Type-Safe**: Full TypeScript coverage

## Notes

- All pages require authentication (except login)
- API base URL defaults to `http://localhost:8000/api`
- Comments and activities auto-update on detail page
- Mobile menu closes automatically on navigation
- Demo credentials shown on login form for easy testing

## Browser Support

- Chrome/Edge (latest 2 versions)
- Firefox (latest 2 versions)
- Safari (latest 2 versions)
- Mobile browsers (iOS Safari, Chrome Mobile)

## Next Steps

1. Run `npm install` to install dependencies
2. Create `.env.local` with your API endpoint
3. Run `npm run dev` to start development
4. Build UI against your backend API
