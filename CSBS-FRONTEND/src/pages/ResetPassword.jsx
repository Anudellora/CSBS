import { useState } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { Lock, Eye, EyeOff } from 'lucide-react';
import { authService } from '../services/api';
import { validatePassword } from '../utils/validators';
import './ResetPassword.css';

export default function ResetPassword() {
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const token = searchParams.get('token') || '';

    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [showConfirm, setShowConfirm] = useState(false);
    const [errors, setErrors] = useState({});
    const [success, setSuccess] = useState(false);
    const [loading, setLoading] = useState(false);

    const validate = () => {
        const newErrors = {};
        const pwdError = validatePassword(password, true);
        if (pwdError) newErrors.password = pwdError;
        if (password !== confirmPassword) newErrors.confirmPassword = 'Пароли не совпадают';
        setErrors(newErrors);
        return Object.keys(newErrors).length === 0;
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        if (!token) {
            setErrors({ submit: 'Недействительная ссылка. Запросите восстановление пароля заново.' });
            return;
        }
        if (!validate()) return;

        setLoading(true);
        setErrors({});
        try {
            await authService.resetPassword(token, password);
            setSuccess(true);
        } catch (err) {
            setErrors({ submit: err.message || 'Ошибка. Попробуйте запросить новую ссылку.' });
        } finally {
            setLoading(false);
        }
    };

    if (success) {
        return (
            <div className="reset-page">
                <div className="reset-card glass-panel">
                    <h2>Пароль изменён</h2>
                    <p className="reset-success-text">
                        Ваш пароль успешно обновлён. Теперь вы можете войти с новым паролем.
                    </p>
                    <button
                        className="btn btn-primary btn-submit"
                        onClick={() => navigate('/')}
                    >
                        На главную
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="reset-page">
            <div className="reset-card glass-panel">
                <h2>Новый пароль</h2>
                <p className="reset-subtitle">Введите новый пароль для вашего аккаунта.</p>

                <form className="auth-form" onSubmit={handleSubmit} noValidate>
                    <div className="form-group">
                        <label>Новый пароль</label>
                        <div className="input-with-icon">
                            <Lock size={18} className="input-icon" />
                            <input
                                type={showPassword ? 'text' : 'password'}
                                placeholder="Минимум 8 символов"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                className={errors.password ? 'input-error' : ''}
                                autoComplete="new-password"
                            />
                            <button
                                type="button"
                                className="password-toggle-btn"
                                onClick={() => setShowPassword(!showPassword)}
                                tabIndex="-1"
                            >
                                {showPassword ? <Eye size={18} /> : <EyeOff size={18} />}
                            </button>
                        </div>
                        {errors.password && <span className="error-text">{errors.password}</span>}
                    </div>

                    <div className="form-group">
                        <label>Подтвердите пароль</label>
                        <div className="input-with-icon">
                            <Lock size={18} className="input-icon" />
                            <input
                                type={showConfirm ? 'text' : 'password'}
                                placeholder="Повторите пароль"
                                value={confirmPassword}
                                onChange={(e) => setConfirmPassword(e.target.value)}
                                className={errors.confirmPassword ? 'input-error' : ''}
                                autoComplete="new-password"
                            />
                            <button
                                type="button"
                                className="password-toggle-btn"
                                onClick={() => setShowConfirm(!showConfirm)}
                                tabIndex="-1"
                            >
                                {showConfirm ? <Eye size={18} /> : <EyeOff size={18} />}
                            </button>
                        </div>
                        {errors.confirmPassword && <span className="error-text">{errors.confirmPassword}</span>}
                    </div>

                    {errors.submit && (
                        <div style={{ color: '#ff4d4f', textAlign: 'center', fontSize: '0.9rem' }}>
                            {errors.submit}
                        </div>
                    )}

                    <button
                        type="submit"
                        className="btn btn-primary btn-submit"
                        disabled={loading}
                    >
                        {loading ? 'Сохранение...' : 'Сохранить пароль'}
                    </button>
                </form>
            </div>
        </div>
    );
}
