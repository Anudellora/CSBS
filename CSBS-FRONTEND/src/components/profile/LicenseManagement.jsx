import { useEffect, useState } from 'react';
import {
    KeyRound, ShieldCheck, ShieldX, RefreshCw, Save, Loader2,
    AlertCircle, CheckCircle2,
} from 'lucide-react';
import { apiService } from '../../services/api';
import './LicenseManagement.css';

// Человекочитаемые подписи известных лимитов; неизвестные показываем как есть.
const LIMIT_LABELS = {
    users: 'Пользователи',
    workspaces: 'Рабочие места',
};

function formatDate(iso) {
    if (!iso) return '—';
    return new Date(iso).toLocaleString('ru-RU', {
        day: '2-digit', month: 'long', year: 'numeric',
        hour: '2-digit', minute: '2-digit',
    });
}

function daysLeft(iso) {
    if (!iso) return null;
    const diff = new Date(iso).getTime() - Date.now();
    return Math.ceil(diff / (1000 * 60 * 60 * 24));
}

export default function LicenseManagement() {
    const [info, setInfo] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    const [token, setToken] = useState('');
    const [installing, setInstalling] = useState(false);
    const [installError, setInstallError] = useState('');
    const [installOk, setInstallOk] = useState(false);

    const load = () => {
        setLoading(true);
        setError('');
        apiService.getLicense()
            .then(setInfo)
            .catch(err => setError(err.message || 'Не удалось загрузить лицензию'))
            .finally(() => setLoading(false));
    };

    useEffect(() => { load(); }, []);

    const handleInstall = async (e) => {
        e.preventDefault();
        if (!token.trim()) return;
        setInstalling(true);
        setInstallError('');
        setInstallOk(false);
        try {
            const updated = await apiService.installLicense(token.trim());
            setInfo(updated);
            setToken('');
            setInstallOk(true);
            setTimeout(() => setInstallOk(false), 3000);
        } catch (err) {
            setInstallError(err.message || 'Лицензия не принята');
        } finally {
            setInstalling(false);
        }
    };

    const left = info?.expires_at ? daysLeft(info.expires_at) : null;
    const expiringSoon = left !== null && left >= 0 && left <= 14;

    return (
        <div className="profile-card glass-panel fade-in">
            <div className="panel-header">
                <h2><KeyRound size={22} className="text-accent" /> Лицензия</h2>
                <div className="panel-toolbar">
                    <button className="btn-ghost" onClick={load} disabled={loading}>
                        <RefreshCw size={16} className={loading ? 'spin' : ''} /> Обновить
                    </button>
                </div>
            </div>

            {loading ? (
                <div className="license-loading">
                    <Loader2 size={20} className="spin" /> Загрузка состояния лицензии…
                </div>
            ) : error ? (
                <div className="license-alert license-alert--error">
                    <AlertCircle size={18} /> {error}
                </div>
            ) : (
                <>
                    {/* ── Текущее состояние ── */}
                    <div className={`license-status ${info.active ? 'is-active' : 'is-inactive'}`}>
                        <div className="license-status-icon">
                            {info.active ? <ShieldCheck size={26} /> : <ShieldX size={26} />}
                        </div>
                        <div className="license-status-text">
                            <span className={`status-badge ${info.active ? 'active' : 'blocked'}`}>
                                {info.active ? 'Лицензия активна' : 'Лицензия неактивна'}
                            </span>
                            {!info.active && info.reason && (
                                <div className="license-reason">{info.reason}</div>
                            )}
                        </div>
                    </div>

                    {/* ── Детали (только если есть данные лицензии) ── */}
                    {(info.customer_id || info.plan) && (
                        <div className="license-grid">
                            <div className="license-row">
                                <span className="license-key">Клиент</span>
                                <span className="license-val">{info.customer_id || '—'}</span>
                            </div>
                            <div className="license-row">
                                <span className="license-key">Тариф</span>
                                <span className="license-val">
                                    {info.plan ? <span className="plan-badge">{info.plan}</span> : '—'}
                                </span>
                            </div>
                            <div className="license-row">
                                <span className="license-key">Действует до</span>
                                <span className="license-val">
                                    {formatDate(info.expires_at)}
                                    {left !== null && left >= 0 && (
                                        <span className={`license-days ${expiringSoon ? 'warn' : ''}`}>
                                            · осталось {left} дн.
                                        </span>
                                    )}
                                </span>
                            </div>
                        </div>
                    )}

                    {/* ── Фичи ── */}
                    {info.features?.length > 0 && (
                        <div className="license-section">
                            <div className="license-section-title">Доступные функции</div>
                            <div className="feature-chips">
                                {info.features.map(f => (
                                    <span key={f} className="feature-chip">
                                        <CheckCircle2 size={14} /> {f}
                                    </span>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* ── Лимиты ── */}
                    {info.limits && Object.keys(info.limits).length > 0 && (
                        <div className="license-section">
                            <div className="license-section-title">Лимиты</div>
                            <div className="limit-cards">
                                {Object.entries(info.limits).map(([key, val]) => (
                                    <div key={key} className="limit-card">
                                        <div className="limit-card-value">{val}</div>
                                        <div className="limit-card-label">{LIMIT_LABELS[key] || key}</div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </>
            )}

            {/* ── Установка нового ключа ── */}
            <form className="license-install" onSubmit={handleInstall}>
                <div className="license-section-title">Установить лицензионный ключ</div>
                <p className="license-hint">
                    Вставьте подписанный токен, выданный вендором. Подпись и срок проверяются
                    на сервере; ключ применяется сразу, без перезапуска.
                </p>
                <textarea
                    className="license-textarea"
                    placeholder="eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9…"
                    value={token}
                    onChange={e => { setToken(e.target.value); setInstallError(''); }}
                    rows={4}
                    spellCheck={false}
                />
                {installError && (
                    <div className="license-alert license-alert--error">
                        <AlertCircle size={18} /> {installError}
                    </div>
                )}
                <div className="license-install-actions">
                    <button className="btn-accent" type="submit" disabled={installing || !token.trim()}>
                        {installing
                            ? <><Loader2 size={16} className="spin" /> Проверка…</>
                            : <><Save size={16} /> {installOk ? 'Установлено ✓' : 'Установить'}</>}
                    </button>
                </div>
            </form>
        </div>
    );
}
