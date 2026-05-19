import { useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Bot, User } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import cowLogo from '../assets/cow.png';
import AuthModal from './AuthModal';
import ThemeToggle from './ThemeToggle';
import StaggeredMenu from './StaggeredMenu/StaggeredMenu';
import './Navigation.css';

export default function Navigation() {
    const location = useLocation();
    const { isLoggedIn, logout } = useAuth();
    const navigate = useNavigate();
    const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
    const [authMode, setAuthMode] = useState('login');

    const handleLogout = async () => {
        await logout();
        navigate('/');
    };

    const openAuthModal = (mode) => {
        setAuthMode(mode);
        setIsAuthModalOpen(true);
    };

    const menuItems = [
        { label: 'Главная', link: '/' },
        { label: 'Бронирование', link: '/booking' },
        { label: 'ИИ-Ассистент', link: '/ai-assistant' },
        ...(isLoggedIn ? [{ label: 'Личный кабинет', link: '/profile' }] : [])
    ];

    const actionItems = isLoggedIn
        ? [{ label: 'Выйти', onClick: handleLogout }]
        : [{ label: 'Вход / Регистрация', onClick: () => openAuthModal('login') }];

    return (
        <nav className="navbar glass-panel">
            <div className="container nav-container">
                <Link to="/" className="nav-logo">
                    <img
                        src={cowLogo}
                        className="logo-icon neon-border"
                        alt="Cow Logo"
                        width="100"
                        height="100"
                        decoding="async"
                        fetchPriority="high"
                    />
                    <span className="brand-name">COW</span>
                </Link>

                {/* Desktop navigation */}
                <div className="nav-links nav-links-desktop">
                    <Link to="/" className={`nav-link ${location.pathname === '/' ? 'active' : ''}`}>
                        Главная
                    </Link>
                    <Link to="/booking" className={`nav-link ${location.pathname === '/booking' ? 'active' : ''}`}>
                        Бронирование
                    </Link>
                    <Link to="/ai-assistant" className={`nav-link nav-link-ai ${location.pathname === '/ai-assistant' ? 'active' : ''}`}>
                        <Bot size={18} />
                        ИИ-Ассистент
                    </Link>

                    {isLoggedIn && (
                        <Link to="/profile" className={`nav-link ${location.pathname === '/profile' ? 'active' : ''}`}>
                            <User size={18} />
                            Личный кабинет
                        </Link>
                    )}

                    {isLoggedIn ? (
                        <button className="btn btn-outline" style={{ borderColor: 'var(--color-border)', color: 'var(--color-text-main)' }} onClick={handleLogout}>
                            Выйти
                        </button>
                    ) : (
                        <button className="btn btn-primary" onClick={() => openAuthModal('login')}>
                            Вход/Регистрация
                        </button>
                    )}

                    <ThemeToggle />
                </div>

                {/* Mobile navigation — StaggeredMenu burger */}
                <div className="nav-links-mobile">
                    <ThemeToggle className="theme-toggle-mobile" />
                    <StaggeredMenu
                        position="right"
                        colors={['#00a6c0', '#222831']}
                        accentColor="#00a6c0"
                        menuButtonColor="#ffffff"
                        openMenuButtonColor="#ffffff"
                        items={menuItems}
                        actionItems={actionItems}
                        displayItemNumbering
                    />
                </div>
            </div>

            <AuthModal
                isOpen={isAuthModalOpen}
                onClose={() => setIsAuthModalOpen(false)}
                initialMode={authMode}
            />
        </nav>
    );
}
