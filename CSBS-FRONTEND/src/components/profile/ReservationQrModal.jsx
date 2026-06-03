import { useEffect, useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { X, Loader2, MapPin, Clock, AlertCircle } from 'lucide-react';
import { apiService } from '../../services/api';
import './ReservationQrModal.css';

export default function ReservationQrModal({ reservationId, onClose }) {
    const [data, setData] = useState(null);
    const [error, setError] = useState('');

    useEffect(() => {
        let cancelled = false;
        apiService.getReservationPass(reservationId)
            .then(d => { if (!cancelled) setData(d); })
            .catch(err => { if (!cancelled) setError(err.message || 'Не удалось получить пропуск'); });
        return () => { cancelled = true; };
    }, [reservationId]);

    // Esc закрывает модалку
    useEffect(() => {
        const onKey = (e) => { if (e.key === 'Escape') onClose(); };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, [onClose]);

    const formatRange = () => {
        if (!data) return '';
        const start = new Date(data.start_time);
        const end = new Date(data.end_time);
        return `${start.toLocaleString('ru-RU')} — ${end.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}`;
    };

    return (
        <div className="qr-modal-backdrop" onClick={onClose}>
            <div className="qr-modal" onClick={(e) => e.stopPropagation()}>
                <button className="qr-modal-close" onClick={onClose} aria-label="Закрыть">
                    <X size={18} />
                </button>

                <h3>Пропуск в коворкинг</h3>
                <p className="qr-modal-subtitle">Покажите QR-код на ресепшене для входа</p>

                <div className="qr-modal-body">
                    {error && (
                        <div className="qr-modal-error">
                            <AlertCircle size={18} />
                            <span>{error}</span>
                        </div>
                    )}

                    {!error && !data && (
                        <div className="qr-modal-loading">
                            <Loader2 className="animate-spin" size={28} />
                            <span>Генерируем пропуск...</span>
                        </div>
                    )}

                    {data && (
                        <>
                            <div className="qr-modal-code">
                                <QRCodeSVG
                                    value={data.token}
                                    size={240}
                                    level="M"
                                    includeMargin={true}
                                />
                            </div>
                            <div className="qr-modal-meta">
                                <div className="qr-modal-meta-row">
                                    <MapPin size={14} />
                                    <span>{data.workspace}{data.location ? ` · ${data.location}` : ''}</span>
                                </div>
                                <div className="qr-modal-meta-row">
                                    <Clock size={14} />
                                    <span>{formatRange()}</span>
                                </div>
                            </div>
                        </>
                    )}
                </div>
            </div>
        </div>
    );
}
