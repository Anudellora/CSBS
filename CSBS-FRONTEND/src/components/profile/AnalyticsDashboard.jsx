import { useEffect, useState } from 'react';
import {
    BarChart, Bar, LineChart, Line, PieChart, Pie, Cell,
    XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, LabelList,
} from 'recharts';
import {
    BarChart3, TrendingUp, Activity, Users, Sparkles, Loader2, AlertCircle, RefreshCw,
} from 'lucide-react';
import { apiService } from '../../services/api';
import './AnalyticsDashboard.css';

// Палитра в стиле сайта (--color-accent + дополнения).
const PALETTE = ['#00a6c0', '#00c4e4', '#7ad6e3', '#3ec5b1', '#5b9bd5', '#a48bd3', '#e2725b'];

const CHART_AXIS = '#959fa8';
const CHART_GRID = 'rgba(216, 215, 206, 0.1)';
const ACCENT = '#00a6c0';

function KpiCard({ icon: Icon, label, value, sub, accent }) {
    return (
        <div className={`kpi-card ${accent ? 'kpi-card--accent' : ''}`}>
            <div className="kpi-card-icon">
                <Icon size={18} />
            </div>
            <div className="kpi-card-body">
                <div className="kpi-card-label">{label}</div>
                <div className="kpi-card-value">{value}</div>
                {sub && <div className="kpi-card-sub">{sub}</div>}
            </div>
        </div>
    );
}

function ChartCard({ title, icon: Icon, hint, children }) {
    return (
        <div className="chart-card glass-panel">
            <div className="chart-card-header">
                <h3>{Icon && <Icon size={16} />} {title}</h3>
                {hint && <span className="chart-card-hint">{hint}</span>}
            </div>
            <div className="chart-card-body">
                {children}
            </div>
        </div>
    );
}

function tooltipStyle() {
    return {
        backgroundColor: 'rgba(34, 40, 49, 0.95)',
        border: '1px solid rgba(216, 215, 206, 0.15)',
        borderRadius: 8,
        color: '#d8d7ce',
        fontSize: 13,
    };
}

export default function AnalyticsDashboard() {
    const [data, setData] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    const load = () => {
        setLoading(true);
        setError('');
        apiService.getAnalyticsDashboard()
            .then(setData)
            .catch(err => setError(err.message || 'Не удалось загрузить аналитику'))
            .finally(() => setLoading(false));
    };

    useEffect(() => { load(); }, []);

    if (loading) {
        return (
            <div className="profile-card glass-panel fade-in analytics-loading">
                <Loader2 className="animate-spin" size={32} />
                <span>Собираем аналитику... LLM-инсайт может занять несколько секунд.</span>
            </div>
        );
    }

    if (error) {
        return (
            <div className="profile-card glass-panel fade-in analytics-error">
                <AlertCircle size={24} />
                <div>
                    <div>{error}</div>
                    <button className="btn-refresh" onClick={load}>
                        <RefreshCw size={14} /> Повторить
                    </button>
                </div>
            </div>
        );
    }

    if (!data) return null;

    const { today, ml_week, actual_last_14, by_category, by_location, llm_today } = data;

    return (
        <div className="analytics-dashboard fade-in">
            <div className="analytics-header">
                <div>
                    <h2><BarChart3 size={22} className="text-accent" /> Аналитика и прогноз</h2>
                    <p className="text-muted">Фактическая загрузка, ML-прогноз на неделю и рекомендации LLM</p>
                </div>
                <button className="btn-refresh" onClick={load} title="Обновить">
                    <RefreshCw size={14} /> Обновить
                </button>
            </div>

            {/* KPI */}
            <div className="kpi-grid">
                <KpiCard
                    icon={Activity}
                    label="Загрузка сегодня"
                    value={`${today.load_percent}%`}
                    sub={`${today.workspaces_active} из ${today.workspaces_total} мест`}
                    accent
                />
                <KpiCard
                    icon={Users}
                    label="Бронирований сегодня"
                    value={today.bookings_count}
                />
                <KpiCard
                    icon={TrendingUp}
                    label="ML-прогноз, ср. за неделю"
                    value={`${avgOf(ml_week)}%`}
                    sub="LightGBM"
                />
            </div>

            {/* LLM Insight */}
            {llm_today?.message && (
                <div className="llm-insight glass-panel">
                    <div className="llm-insight-header">
                        <Sparkles size={16} />
                        <span>Совет AI-менеджера на сегодня</span>
                    </div>
                    <p>{llm_today.message}</p>
                </div>
            )}

            {/* Charts grid */}
            <div className="charts-grid">
                <ChartCard title="ML-прогноз загрузки на неделю" icon={TrendingUp} hint="LightGBM, %">
                    <ResponsiveContainer width="100%" height={260}>
                        <BarChart data={ml_week} margin={{ top: 16, right: 8, bottom: 0, left: -16 }}>
                            <CartesianGrid stroke={CHART_GRID} strokeDasharray="3 3" vertical={false} />
                            <XAxis dataKey="label" stroke={CHART_AXIS} tick={{ fontSize: 12 }} />
                            <YAxis stroke={CHART_AXIS} tick={{ fontSize: 12 }} domain={[0, 100]} unit="%" />
                            <Tooltip
                                cursor={{ fill: 'rgba(0, 166, 192, 0.08)' }}
                                contentStyle={tooltipStyle()}
                                formatter={(v) => [`${v}%`, 'Прогноз']}
                            />
                            <Bar dataKey="load_percent" fill={ACCENT} radius={[6, 6, 0, 0]}>
                                <LabelList dataKey="load_percent" position="top" fill={CHART_AXIS} fontSize={11} formatter={(v) => `${v}%`} />
                            </Bar>
                        </BarChart>
                    </ResponsiveContainer>
                </ChartCard>

                <ChartCard title="Фактическая загрузка за 14 дней" icon={Activity} hint="по уникальным местам">
                    <ResponsiveContainer width="100%" height={260}>
                        <LineChart data={actual_last_14} margin={{ top: 16, right: 16, bottom: 0, left: -16 }}>
                            <CartesianGrid stroke={CHART_GRID} strokeDasharray="3 3" vertical={false} />
                            <XAxis dataKey="label" stroke={CHART_AXIS} tick={{ fontSize: 11 }} />
                            <YAxis stroke={CHART_AXIS} tick={{ fontSize: 12 }} domain={[0, 100]} unit="%" />
                            <Tooltip
                                cursor={{ stroke: ACCENT, strokeOpacity: 0.4 }}
                                contentStyle={tooltipStyle()}
                                formatter={(v) => [`${v}%`, 'Загрузка']}
                            />
                            <Line
                                type="monotone"
                                dataKey="load_percent"
                                stroke={ACCENT}
                                strokeWidth={2.5}
                                dot={{ fill: ACCENT, r: 3 }}
                                activeDot={{ r: 6 }}
                            />
                        </LineChart>
                    </ResponsiveContainer>
                </ChartCard>

                <ChartCard title="Брони по категориям" icon={Users} hint="за 30 дней">
                    {by_category.length === 0 ? (
                        <EmptyState text="Нет броней за последние 30 дней" />
                    ) : (
                        <ResponsiveContainer width="100%" height={260}>
                            <PieChart>
                                <Tooltip
                                    contentStyle={tooltipStyle()}
                                    formatter={(v, name) => [`${v}`, name]}
                                />
                                <Legend
                                    verticalAlign="bottom"
                                    iconType="circle"
                                    wrapperStyle={{ fontSize: 12, color: CHART_AXIS }}
                                />
                                <Pie
                                    data={by_category}
                                    dataKey="count"
                                    nameKey="category"
                                    innerRadius={55}
                                    outerRadius={85}
                                    paddingAngle={2}
                                >
                                    {by_category.map((_, i) => (
                                        <Cell key={i} fill={PALETTE[i % PALETTE.length]} stroke="transparent" />
                                    ))}
                                </Pie>
                            </PieChart>
                        </ResponsiveContainer>
                    )}
                </ChartCard>

                <ChartCard title="Брони по локациям" icon={Users} hint="за 30 дней">
                    {by_location.length === 0 ? (
                        <EmptyState text="Нет броней за последние 30 дней" />
                    ) : (
                        <ResponsiveContainer width="100%" height={260}>
                            <BarChart data={by_location} layout="vertical" margin={{ top: 8, right: 16, bottom: 0, left: 8 }}>
                                <CartesianGrid stroke={CHART_GRID} strokeDasharray="3 3" horizontal={false} />
                                <XAxis type="number" stroke={CHART_AXIS} tick={{ fontSize: 12 }} />
                                <YAxis type="category" dataKey="location" stroke={CHART_AXIS} tick={{ fontSize: 12 }} width={140} />
                                <Tooltip
                                    cursor={{ fill: 'rgba(0, 166, 192, 0.08)' }}
                                    contentStyle={tooltipStyle()}
                                    formatter={(v) => [`${v}`, 'Броней']}
                                />
                                <Bar dataKey="count" fill={ACCENT} radius={[0, 6, 6, 0]} />
                            </BarChart>
                        </ResponsiveContainer>
                    )}
                </ChartCard>
            </div>
        </div>
    );
}

function EmptyState({ text }) {
    return <div className="chart-empty">{text}</div>;
}

function avgOf(list) {
    if (!list || list.length === 0) return 0;
    const sum = list.reduce((acc, x) => acc + (x.load_percent || 0), 0);
    return Math.round(sum / list.length);
}
