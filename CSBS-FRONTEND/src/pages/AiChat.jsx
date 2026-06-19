import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Bot, Send, User, Loader2, CalendarCheck, Clock, MapPin, LogIn, Lock, Users, Building2, Cpu } from 'lucide-react';
import { apiService } from '../services/api';
import { categoryToBookingType } from '../utils/formatters';
import AuthModal from '../components/AuthModal';
import { useAuth } from '../context/AuthContext';
import './AiChat.css';


function LocationsGrid({ locations, onPick, disabled }) {
    if (!locations || locations.length === 0) return null;
    return (
        <div className="workspaces-grid">
            {locations.map((loc) => (
                <div key={loc.id} className="workspace-card">
                    <div className="workspace-card-header">
                        <Building2 size={16} />
                        <span className="workspace-card-name">{loc.name}</span>
                    </div>
                    <div className="workspace-card-body">
                        {loc.address && (
                            <div className="workspace-card-row">
                                <MapPin size={14} />
                                <span>{loc.address}</span>
                            </div>
                        )}
                    </div>
                    <button
                        type="button"
                        className="workspace-card-cta"
                        onClick={() => onPick(loc)}
                        disabled={disabled}
                    >
                        Показать места
                    </button>
                </div>
            ))}
        </div>
    );
}

function WorkspacesGrid({ workspaces, onPick, disabled }) {
    if (!workspaces || workspaces.length === 0) return null;
    return (
        <div className="workspaces-grid">
            {workspaces.map((ws) => (
                <div key={ws.id} className="workspace-card">
                    <div className="workspace-card-header">
                        <MapPin size={16} />
                        <span className="workspace-card-name">{ws.name}</span>
                        {ws.category && (
                            <span className="workspace-card-badge">{ws.category}</span>
                        )}
                    </div>
                    <div className="workspace-card-body">
                        {ws.location_name && (
                            <div className="workspace-card-row">
                                <Building2 size={14} />
                                <span>{ws.location_name}</span>
                            </div>
                        )}
                        {ws.capacity > 0 && (
                            <div className="workspace-card-row">
                                <Users size={14} />
                                <span>до {ws.capacity} чел.</span>
                            </div>
                        )}
                    </div>
                    <button
                        type="button"
                        className="workspace-card-cta"
                        onClick={() => onPick(ws)}
                        disabled={disabled}
                    >
                        Забронировать
                    </button>
                </div>
            ))}
        </div>
    );
}

function BookingCard({ booking }) {
    return (
        <div className="booking-card">
            <div className="booking-card-header">
                <CalendarCheck size={18} />
                <span>Бронирование подтверждено!</span>
            </div>
            <div className="booking-card-body">
                <div className="booking-card-row">
                    <MapPin size={14} />
                    <span>{booking.workspace_name}</span>
                </div>
                <div className="booking-card-row">
                    <CalendarCheck size={14} />
                    <span>{booking.date}</span>
                </div>
                <div className="booking-card-row">
                    <Clock size={14} />
                    <span>{booking.time_from} — {booking.time_to}</span>
                </div>
                {booking.price && (
                    <div className="booking-card-price">
                        {booking.price}
                    </div>
                )}
            </div>
        </div>
    );
}

export default function AiChat() {
    const [messages, setMessages] = useState(() => {
        const saved = sessionStorage.getItem('ai_chat_history');
        if (saved) {
            try {
                return JSON.parse(saved);
            } catch (e) {
                console.error("Failed to parse ai_chat_history", e);
            }
        }
        return [
            { role: 'ai', content: 'Здравствуйте! Я ваш ИИ-помощник. Ищете идеальное рабочее место, хотите проверить доступность или узнать прогноз цен на ближайшие даты? Я также могу забронировать место для вас прямо здесь!' }
        ];
    });
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
    const [models, setModels] = useState([]);
    const [selectedModel, setSelectedModel] = useState(() => {
        return localStorage.getItem('ai_chat_model') || 'gigachat';
    });
    const messagesContainerRef = useRef(null);
    const lastMessageRef = useRef(null);
    const { isLoggedIn } = useAuth();
    const navigate = useNavigate();

    useEffect(() => {
        apiService.getAiModels()
            .then(list => {
                setModels(list);
                // Если сохранённая модель недоступна на сервере — переключаемся на первую доступную
                const current = list.find(m => m.id === selectedModel);
                if (!current?.available) {
                    const firstAvailable = list.find(m => m.available);
                    if (firstAvailable) {
                        setSelectedModel(firstAvailable.id);
                        localStorage.setItem('ai_chat_model', firstAvailable.id);
                    }
                }
            })
            .catch(err => console.error('Failed to load AI models', err));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const handleModelChange = (modelId) => {
        setSelectedModel(modelId);
        localStorage.setItem('ai_chat_model', modelId);
    };

    const scrollToBottom = () => {
        const el = messagesContainerRef.current;
        if (el) {
            el.scrollTop = el.scrollHeight;
        }
    };

    useEffect(() => {
        sessionStorage.setItem('ai_chat_history', JSON.stringify(messages));

        const lastMsg = messages[messages.length - 1];
        const isCardList = lastMsg?.action === 'list_workspaces' || lastMsg?.action === 'list_locations';

        if (isCardList && lastMessageRef.current && messagesContainerRef.current) {
            // Скроллим только внутри контейнера, не затрагивая страницу
            const container = messagesContainerRef.current;
            const msgEl = lastMessageRef.current;
            const offset = msgEl.offsetTop - container.offsetTop;
            container.scrollTo({ top: offset, behavior: 'smooth' });
        } else {
            scrollToBottom();
        }
    }, [messages, isLoading]);

    const handleSend = async (overrideMessage) => {
        const userMessage = (typeof overrideMessage === 'string' ? overrideMessage : input).trim();
        if (!userMessage || isLoading || !isLoggedIn) return;

        const newMessages = [...messages, { role: 'user', content: userMessage }];
        setMessages(newMessages);
        if (typeof overrideMessage !== 'string') setInput('');
        setIsLoading(true);

        try {
            // Отправляем историю (без первого приветственного сообщения, если нужно)
            const history = newMessages
                .filter((_, i) => i > 0) // пропускаем стартовое приветствие
                .slice(-10) // последние 10 сообщений
                .map(m => ({ role: m.role === 'ai' ? 'assistant' : 'user', content: m.content }));

            const data = await apiService.sendAiMessage(userMessage, history, selectedModel);

            const aiMessage = {
                role: 'ai',
                content: data.reply,
                action: data.action || null,
                booking: data.booking || null,
                workspaces: data.workspaces || null,
                locations: data.locations || null,
            };

            setMessages(msgs => [...msgs, aiMessage]);
        } catch (error) {
            console.error(error);
            // 402 → функция AI-чата недоступна по текущей лицензии: показываем
            // понятную причину с сервера вместо общей ошибки соединения.
            const content = error.status === 402
                ? (error.message || 'AI-чат недоступен в вашей лицензии.')
                : 'Ошибка соединения с сервером.';
            setMessages(msgs => [...msgs, { role: 'ai', content }]);
        } finally {
            setIsLoading(false);
        }
    };

    const handlePickWorkspace = (ws) => {
        navigate('/booking', {
            state: {
                workspaceName: ws.name,
                locationId: ws.location_id ? String(ws.location_id) : '',
                bookingType: categoryToBookingType(ws.category),
            },
        });
    };

    const handlePickLocation = (loc) => {
        handleSend(`Покажи доступные места в локации "${loc.name}"`);
    };

    const handleKeyDown = (e) => {
        if (e.key === 'Enter') handleSend();
    };

    return (
        <div className="chat-page container">
            <div className="chat-header">
                <div className="ai-avatar">
                    <Bot size={32} className="text-accent" />
                </div>
                <div className="chat-header-info">
                    <h2>ИИ-Ассистент</h2>
                    <p className="text-muted">Прогнозирование цен, подбор и бронирование рабочего пространства</p>
                </div>
                {models.length > 0 && (
                    <div className="model-switcher" role="radiogroup" aria-label="Выбор модели ИИ">
                        <div className="model-switcher-label">
                            <Cpu size={14} />
                            <span>Модель</span>
                        </div>
                        <div className="model-switcher-options">
                            {models.map(m => (
                                <button
                                    key={m.id}
                                    type="button"
                                    role="radio"
                                    aria-checked={selectedModel === m.id}
                                    className={`model-option ${selectedModel === m.id ? 'active' : ''}`}
                                    onClick={() => handleModelChange(m.id)}
                                    disabled={!m.available || isLoading}
                                    title={!m.available ? 'Эта модель не настроена на сервере' : ''}
                                >
                                    <span className="model-option-label">{m.label}</span>
                                    <span className="model-option-origin">{m.origin}</span>
                                </button>
                            ))}
                        </div>
                    </div>
                )}
            </div>

            <div className="chat-container glass-panel" style={{ position: 'relative' }}>
                {!isLoggedIn && (
                    <div
                        className="auth-overlay"
                        style={{
                            position: 'absolute',
                            top: 0, left: 0, right: 0, bottom: 0,
                            backgroundColor: 'rgba(10, 14, 23, 0.75)',
                            backdropFilter: 'blur(4px)',
                            zIndex: 50,
                            display: 'flex',
                            flexDirection: 'column',
                            justifyContent: 'center',
                            alignItems: 'center',
                            borderRadius: '16px',
                            color: '#fff',
                            textAlign: 'center',
                            padding: '2rem',
                        }}
                    >
                        <Lock size={64} style={{ marginBottom: '1.5rem', color: 'var(--color-accent)' }} />
                        <h3 style={{ fontSize: '1.75rem', marginBottom: '1rem', fontWeight: 600 }}>Требуется авторизация</h3>
                        <p style={{ color: 'var(--color-text-muted)', fontSize: '1.1rem', marginBottom: '2rem', maxWidth: '400px' }}>
                            Войдите или зарегистрируйтесь, чтобы общаться с ИИ-ассистентом и бронировать места прямо в чате
                        </p>
                        <button className="btn btn-primary btn-lg" onClick={() => setIsAuthModalOpen(true)}>
                            Вход/Регистрация
                        </button>
                    </div>
                )}

                <div style={{ display: 'contents', opacity: isLoggedIn ? 1 : 0.4, pointerEvents: isLoggedIn ? 'auto' : 'none' }}>
                <div className="chat-messages" ref={messagesContainerRef}>
                    {messages.map((msg, idx) => (
                        <div
                            key={idx}
                            ref={idx === messages.length - 1 ? lastMessageRef : null}
                            className={`message-wrapper ${msg.role}`}
                        >
                            <div className="message-bubble">
                                {msg.role === 'ai' && <Bot size={16} className="message-icon" />}
                                {msg.role === 'user' && <User size={16} className="message-icon" />}
                                <div className="message-content">
                                    <p>{msg.content}</p>

                                    {/* Карточка бронирования */}
                                    {msg.action === 'booked' && msg.booking && (
                                        <BookingCard booking={msg.booking} />
                                    )}

                                    {/* Список локаций */}
                                    {msg.action === 'list_locations' && msg.locations && (
                                        <LocationsGrid
                                            locations={msg.locations}
                                            onPick={handlePickLocation}
                                            disabled={isLoading || !isLoggedIn}
                                        />
                                    )}

                                    {/* Список доступных мест */}
                                    {msg.action === 'list_workspaces' && msg.workspaces && (
                                        <WorkspacesGrid
                                            workspaces={msg.workspaces}
                                            onPick={handlePickWorkspace}
                                            disabled={isLoading || !isLoggedIn}
                                        />
                                    )}

                                    {/* Кнопка авторизации */}
                                    {msg.action === 'need_auth' && (
                                        <button
                                            className="chat-auth-btn"
                                            onClick={() => setIsAuthModalOpen(true)}
                                        >
                                            <LogIn size={16} />
                                            Войти в аккаунт
                                        </button>
                                    )}
                                </div>
                            </div>
                        </div>
                    ))}
                    {isLoading && (
                        <div className="message-wrapper ai">
                            <div className="message-bubble">
                                <Bot size={16} className="message-icon" />
                                <Loader2 className="animate-spin" size={16} />
                            </div>
                        </div>
                    )}
                </div>

                <div className="chat-input-area">
                    <input
                        type="text"
                        className="chat-input"
                        placeholder="Введите ваше сообщение..."
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        onKeyDown={handleKeyDown}
                        disabled={isLoading || !isLoggedIn}
                    />
                    <button className="btn-send" onClick={handleSend} disabled={isLoading || !isLoggedIn}>
                        <Send size={20} />
                    </button>
                </div>
                </div>
            </div>

            <AuthModal
                isOpen={isAuthModalOpen}
                onClose={() => setIsAuthModalOpen(false)}
                initialMode="login"
            />
        </div>
    );
}
