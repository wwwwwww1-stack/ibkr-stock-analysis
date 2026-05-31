export namespace domain {

	export class AccountPosition {
	    symbol: string;
	    quantity: number;
	    average_cost: number;
	    market_price: number;
	    market_value_usd: number;
	    unrealized_pnl_usd: number;

	    static createFrom(source: any = {}) {
	        return new AccountPosition(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.quantity = source["quantity"];
	        this.average_cost = source["average_cost"];
	        this.market_price = source["market_price"];
	        this.market_value_usd = source["market_value_usd"];
	        this.unrealized_pnl_usd = source["unrealized_pnl_usd"];
	    }
	}
	export class AccountPositionContext {
	    symbol: string;
	    quantity: number;
	    average_cost: number;
	    market_price: number;
	    market_value_usd: number;
	    unrealized_pnl_usd: number;

	    static createFrom(source: any = {}) {
	        return new AccountPositionContext(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.quantity = source["quantity"];
	        this.average_cost = source["average_cost"];
	        this.market_price = source["market_price"];
	        this.market_value_usd = source["market_value_usd"];
	        this.unrealized_pnl_usd = source["unrealized_pnl_usd"];
	    }
	}
	export class AccountSnapshot {
	    available_cash_usd: number;
	    buying_power_usd: number;
	    // Go type: time
	    snapshot_at: any;
	    positions: AccountPosition[];
	    notes?: string[];

	    static createFrom(source: any = {}) {
	        return new AccountSnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available_cash_usd = source["available_cash_usd"];
	        this.buying_power_usd = source["buying_power_usd"];
	        this.snapshot_at = this.convertValues(source["snapshot_at"], null);
	        this.positions = this.convertValues(source["positions"], AccountPosition);
	        this.notes = source["notes"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SizingEnvelope {
	    symbol: string;
	    sizing_status: string;
	    account_notional_available_usd: number;
	    user_notional_cap_usd: number;
	    new_exposure_cap_usd: number;
	    existing_symbol_exposure_usd: number;
	    remaining_symbol_cap_usd: number;
	    advisory_notional_cap_usd: number;
	    reference_entry_price?: number;
	    advisory_max_shares?: number;
	    risk_per_share?: number;
	    estimated_risk_usd?: number;

	    static createFrom(source: any = {}) {
	        return new SizingEnvelope(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.sizing_status = source["sizing_status"];
	        this.account_notional_available_usd = source["account_notional_available_usd"];
	        this.user_notional_cap_usd = source["user_notional_cap_usd"];
	        this.new_exposure_cap_usd = source["new_exposure_cap_usd"];
	        this.existing_symbol_exposure_usd = source["existing_symbol_exposure_usd"];
	        this.remaining_symbol_cap_usd = source["remaining_symbol_cap_usd"];
	        this.advisory_notional_cap_usd = source["advisory_notional_cap_usd"];
	        this.reference_entry_price = source["reference_entry_price"];
	        this.advisory_max_shares = source["advisory_max_shares"];
	        this.risk_per_share = source["risk_per_share"];
	        this.estimated_risk_usd = source["estimated_risk_usd"];
	    }
	}
	export class AccountSnapshotContext {
	    available_cash_usd: number;
	    buying_power_usd: number;
	    // Go type: time
	    snapshot_at: any;
	    positions: AccountPositionContext[];
	    max_stock_trade_amount_usd: number;
	    sizing_envelope?: SizingEnvelope;

	    static createFrom(source: any = {}) {
	        return new AccountSnapshotContext(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available_cash_usd = source["available_cash_usd"];
	        this.buying_power_usd = source["buying_power_usd"];
	        this.snapshot_at = this.convertValues(source["snapshot_at"], null);
	        this.positions = this.convertValues(source["positions"], AccountPositionContext);
	        this.max_stock_trade_amount_usd = source["max_stock_trade_amount_usd"];
	        this.sizing_envelope = this.convertValues(source["sizing_envelope"], SizingEnvelope);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AccountSnapshotState {
	    status: string;
	    snapshot?: AccountSnapshot;
	    error?: string;
	    // Go type: time
	    updated_at?: any;

	    static createFrom(source: any = {}) {
	        return new AccountSnapshotState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.snapshot = this.convertValues(source["snapshot"], AccountSnapshot);
	        this.error = source["error"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PositionManagementOutput {
	    account_aware: boolean;
	    sizing_status: string;
	    advisory_action: string;
	    advisory_max_shares?: number;
	    advisory_notional_cap_usd?: number;
	    estimated_risk_usd?: number;
	    existing_exposure_usd?: number;
	    management_notes: string[];
	    manual_review_required: boolean;

	    static createFrom(source: any = {}) {
	        return new PositionManagementOutput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_aware = source["account_aware"];
	        this.sizing_status = source["sizing_status"];
	        this.advisory_action = source["advisory_action"];
	        this.advisory_max_shares = source["advisory_max_shares"];
	        this.advisory_notional_cap_usd = source["advisory_notional_cap_usd"];
	        this.estimated_risk_usd = source["estimated_risk_usd"];
	        this.existing_exposure_usd = source["existing_exposure_usd"];
	        this.management_notes = source["management_notes"];
	        this.manual_review_required = source["manual_review_required"];
	    }
	}
	export class EntryZone {
	    low: number;
	    high: number;

	    static createFrom(source: any = {}) {
	        return new EntryZone(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.low = source["low"];
	        this.high = source["high"];
	    }
	}
	export class AgentOutput {
	    direction: string;
	    setup_quality: string;
	    entry_zone?: EntryZone;
	    stop_loss?: number;
	    take_profit: number[];
	    risk_reward?: number;
	    confidence: number;
	    market_regime: string;
	    trade_thesis: string;
	    counterargument: string;
	    no_trade_reason: string;
	    rejection_reasons: string[];
	    summary: string;
	    price_action: string[];
	    invalidated_if: string;
	    // Go type: time
	    generated_at: any;
	    position_management?: PositionManagementOutput;

	    static createFrom(source: any = {}) {
	        return new AgentOutput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.direction = source["direction"];
	        this.setup_quality = source["setup_quality"];
	        this.entry_zone = this.convertValues(source["entry_zone"], EntryZone);
	        this.stop_loss = source["stop_loss"];
	        this.take_profit = source["take_profit"];
	        this.risk_reward = source["risk_reward"];
	        this.confidence = source["confidence"];
	        this.market_regime = source["market_regime"];
	        this.trade_thesis = source["trade_thesis"];
	        this.counterargument = source["counterargument"];
	        this.no_trade_reason = source["no_trade_reason"];
	        this.rejection_reasons = source["rejection_reasons"];
	        this.summary = source["summary"];
	        this.price_action = source["price_action"];
	        this.invalidated_if = source["invalidated_if"];
	        this.generated_at = this.convertValues(source["generated_at"], null);
	        this.position_management = this.convertValues(source["position_management"], PositionManagementOutput);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DerivedFeatures {
	    session_high: number;
	    session_low: number;
	    recent_swing_highs: number[];
	    recent_swing_lows: number[];
	    atr: number;
	    volume_context: string;
	    last_close_relative_to_range: string;

	    static createFrom(source: any = {}) {
	        return new DerivedFeatures(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_high = source["session_high"];
	        this.session_low = source["session_low"];
	        this.recent_swing_highs = source["recent_swing_highs"];
	        this.recent_swing_lows = source["recent_swing_lows"];
	        this.atr = source["atr"];
	        this.volume_context = source["volume_context"];
	        this.last_close_relative_to_range = source["last_close_relative_to_range"];
	    }
	}
	export class TimeframeContextSummary {
	    timeframe: string;
	    available: boolean;
	    error?: string;
	    current_price?: number;
	    bar_count: number;
	    // Go type: time
	    last_closed_bar_time?: any;
	    derived?: DerivedFeatures;

	    static createFrom(source: any = {}) {
	        return new TimeframeContextSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeframe = source["timeframe"];
	        this.available = source["available"];
	        this.error = source["error"];
	        this.current_price = source["current_price"];
	        this.bar_count = source["bar_count"];
	        this.last_closed_bar_time = this.convertValues(source["last_closed_bar_time"], null);
	        this.derived = this.convertValues(source["derived"], DerivedFeatures);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AnalysisResult {
	    symbol: string;
	    timeframe: string;
	    current_price: number;
	    output: AgentOutput;
	    context_summaries?: TimeframeContextSummary[];
	    stale: boolean;
	    error?: string;
	    // Go type: time
	    updated_at: any;

	    static createFrom(source: any = {}) {
	        return new AnalysisResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.timeframe = source["timeframe"];
	        this.current_price = source["current_price"];
	        this.output = this.convertValues(source["output"], AgentOutput);
	        this.context_summaries = this.convertValues(source["context_summaries"], TimeframeContextSummary);
	        this.stale = source["stale"];
	        this.error = source["error"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AnalysisHistoryRecord {
	    id: number;
	    result: AnalysisResult;

	    static createFrom(source: any = {}) {
	        return new AnalysisHistoryRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.result = this.convertValues(source["result"], AnalysisResult);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AnalysisHistoryPage {
	    records: AnalysisHistoryRecord[];
	    total: number;
	    limit: number;
	    offset: number;

	    static createFrom(source: any = {}) {
	        return new AnalysisHistoryPage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.records = this.convertValues(source["records"], AnalysisHistoryRecord);
	        this.total = source["total"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AnalysisHistoryQuery {
	    symbol?: string;
	    timeframe?: string;
	    direction?: string;
	    limit: number;
	    offset: number;

	    static createFrom(source: any = {}) {
	        return new AnalysisHistoryQuery(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.timeframe = source["timeframe"];
	        this.direction = source["direction"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}


	export class SymbolState {
	    symbol: string;
	    market_data_status: string;
	    // Go type: time
	    last_closed_bar_time?: any;
	    // Go type: time
	    last_analysis_time?: any;
	    job_status: string;
	    current_price?: number;
	    result?: AnalysisResult;
	    account_context?: AccountSnapshotContext;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new SymbolState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.market_data_status = source["market_data_status"];
	        this.last_closed_bar_time = this.convertValues(source["last_closed_bar_time"], null);
	        this.last_analysis_time = this.convertValues(source["last_analysis_time"], null);
	        this.job_status = source["job_status"];
	        this.current_price = source["current_price"];
	        this.result = this.convertValues(source["result"], AnalysisResult);
	        this.account_context = this.convertValues(source["account_context"], AccountSnapshotContext);
	        this.error = source["error"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChartWindow {
	    id: number;
	    app_name: string;
	    title?: string;

	    static createFrom(source: any = {}) {
	        return new ChartWindow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.app_name = source["app_name"];
	        this.title = source["title"];
	    }
	}
	export class Settings {
	    ibkr_host: string;
	    ibkr_port: number;
	    ibkr_client_id: number;
	    watchlist: string[];
	    selected_timeframe: string;
	    chart_window?: ChartWindow;
	    max_stock_trade_amount_usd?: number;

	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ibkr_host = source["ibkr_host"];
	        this.ibkr_port = source["ibkr_port"];
	        this.ibkr_client_id = source["ibkr_client_id"];
	        this.watchlist = source["watchlist"];
	        this.selected_timeframe = source["selected_timeframe"];
	        this.chart_window = this.convertValues(source["chart_window"], ChartWindow);
	        this.max_stock_trade_amount_usd = source["max_stock_trade_amount_usd"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppState {
	    settings: Settings;
	    connection_status: string;
	    scheduled_analysis_enabled: boolean;
	    symbols: SymbolState[];
	    account_snapshot: AccountSnapshotState;
	    last_error?: string;

	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], Settings);
	        this.connection_status = source["connection_status"];
	        this.scheduled_analysis_enabled = source["scheduled_analysis_enabled"];
	        this.symbols = this.convertValues(source["symbols"], SymbolState);
	        this.account_snapshot = this.convertValues(source["account_snapshot"], AccountSnapshotState);
	        this.last_error = source["last_error"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BacktestExitLeg {
	    // Go type: time
	    time: any;
	    price: number;
	    reason: string;
	    shares: number;
	    gross_pnl: number;
	    commission: number;
	    net_pnl: number;
	    return_pct: number;

	    static createFrom(source: any = {}) {
	        return new BacktestExitLeg(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = this.convertValues(source["time"], null);
	        this.price = source["price"];
	        this.reason = source["reason"];
	        this.shares = source["shares"];
	        this.gross_pnl = source["gross_pnl"];
	        this.commission = source["commission"];
	        this.net_pnl = source["net_pnl"];
	        this.return_pct = source["return_pct"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BacktestOpenPosition {
	    symbol: string;
	    direction: string;
	    setup_quality: string;
	    // Go type: time
	    entry_time: any;
	    entry_price: number;
	    // Go type: time
	    mark_time: any;
	    mark_price: number;
	    shares: number;
	    remaining_shares: number;
	    initial_shares: number;
	    partial_taken: boolean;
	    initial_stop_loss: number;
	    stop_loss: number;
	    take_profit: number;
	    realized_gross_pnl: number;
	    realized_commission: number;
	    realized_net_pnl: number;
	    unrealized_gross_pnl: number;
	    commission: number;
	    unrealized_net_pnl: number;
	    unrealized_return_pct: number;
	    // Go type: time
	    signal_time: any;
	    market_regime: string;
	    trade_thesis: string;
	    counterargument: string;
	    no_trade_reason: string;
	    rejection_reasons: string[];
	    summary: string;
	    confidence: number;
	    invalidated_if: string;

	    static createFrom(source: any = {}) {
	        return new BacktestOpenPosition(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.direction = source["direction"];
	        this.setup_quality = source["setup_quality"];
	        this.entry_time = this.convertValues(source["entry_time"], null);
	        this.entry_price = source["entry_price"];
	        this.mark_time = this.convertValues(source["mark_time"], null);
	        this.mark_price = source["mark_price"];
	        this.shares = source["shares"];
	        this.remaining_shares = source["remaining_shares"];
	        this.initial_shares = source["initial_shares"];
	        this.partial_taken = source["partial_taken"];
	        this.initial_stop_loss = source["initial_stop_loss"];
	        this.stop_loss = source["stop_loss"];
	        this.take_profit = source["take_profit"];
	        this.realized_gross_pnl = source["realized_gross_pnl"];
	        this.realized_commission = source["realized_commission"];
	        this.realized_net_pnl = source["realized_net_pnl"];
	        this.unrealized_gross_pnl = source["unrealized_gross_pnl"];
	        this.commission = source["commission"];
	        this.unrealized_net_pnl = source["unrealized_net_pnl"];
	        this.unrealized_return_pct = source["unrealized_return_pct"];
	        this.signal_time = this.convertValues(source["signal_time"], null);
	        this.market_regime = source["market_regime"];
	        this.trade_thesis = source["trade_thesis"];
	        this.counterargument = source["counterargument"];
	        this.no_trade_reason = source["no_trade_reason"];
	        this.rejection_reasons = source["rejection_reasons"];
	        this.summary = source["summary"];
	        this.confidence = source["confidence"];
	        this.invalidated_if = source["invalidated_if"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BacktestSkippedSetup {
	    // Go type: time
	    time: any;
	    direction: string;
	    setup_quality: string;
	    reason: string;
	    market_regime: string;
	    trade_thesis: string;
	    counterargument: string;
	    no_trade_reason: string;
	    rejection_reasons: string[];
	    summary: string;
	    confidence: number;

	    static createFrom(source: any = {}) {
	        return new BacktestSkippedSetup(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = this.convertValues(source["time"], null);
	        this.direction = source["direction"];
	        this.setup_quality = source["setup_quality"];
	        this.reason = source["reason"];
	        this.market_regime = source["market_regime"];
	        this.trade_thesis = source["trade_thesis"];
	        this.counterargument = source["counterargument"];
	        this.no_trade_reason = source["no_trade_reason"];
	        this.rejection_reasons = source["rejection_reasons"];
	        this.summary = source["summary"];
	        this.confidence = source["confidence"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BacktestTrade {
	    symbol: string;
	    direction: string;
	    setup_quality: string;
	    // Go type: time
	    entry_time: any;
	    entry_price: number;
	    // Go type: time
	    exit_time: any;
	    exit_price: number;
	    exit_reason: string;
	    exit_legs: BacktestExitLeg[];
	    shares: number;
	    initial_stop_loss: number;
	    stop_loss: number;
	    take_profit: number;
	    gross_pnl: number;
	    commission: number;
	    net_pnl: number;
	    return_pct: number;
	    // Go type: time
	    signal_time: any;
	    market_regime: string;
	    trade_thesis: string;
	    counterargument: string;
	    no_trade_reason: string;
	    rejection_reasons: string[];
	    summary: string;
	    confidence: number;
	    invalidated_if: string;

	    static createFrom(source: any = {}) {
	        return new BacktestTrade(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.direction = source["direction"];
	        this.setup_quality = source["setup_quality"];
	        this.entry_time = this.convertValues(source["entry_time"], null);
	        this.entry_price = source["entry_price"];
	        this.exit_time = this.convertValues(source["exit_time"], null);
	        this.exit_price = source["exit_price"];
	        this.exit_reason = source["exit_reason"];
	        this.exit_legs = this.convertValues(source["exit_legs"], BacktestExitLeg);
	        this.shares = source["shares"];
	        this.initial_stop_loss = source["initial_stop_loss"];
	        this.stop_loss = source["stop_loss"];
	        this.take_profit = source["take_profit"];
	        this.gross_pnl = source["gross_pnl"];
	        this.commission = source["commission"];
	        this.net_pnl = source["net_pnl"];
	        this.return_pct = source["return_pct"];
	        this.signal_time = this.convertValues(source["signal_time"], null);
	        this.market_regime = source["market_regime"];
	        this.trade_thesis = source["trade_thesis"];
	        this.counterargument = source["counterargument"];
	        this.no_trade_reason = source["no_trade_reason"];
	        this.rejection_reasons = source["rejection_reasons"];
	        this.summary = source["summary"];
	        this.confidence = source["confidence"];
	        this.invalidated_if = source["invalidated_if"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BacktestReport {
	    symbol: string;
	    date: string;
	    start_date: string;
	    end_date: string;
	    timeframe: string;
	    share_quantity: number;
	    slippage_per_share: number;
	    commission_per_order: number;
	    // Go type: time
	    start_time: any;
	    // Go type: time
	    end_time: any;
	    bar_count: number;
	    trade_count: number;
	    winning_trades: number;
	    losing_trades: number;
	    win_rate_pct: number;
	    total_gross_pnl: number;
	    total_commission: number;
	    total_net_pnl: number;
	    total_return_pct: number;
	    max_drawdown: number;
	    trades: BacktestTrade[];
	    skipped_setups: BacktestSkippedSetup[];
	    skip_reason_counts: Record<string, number>;
	    open_position?: BacktestOpenPosition;
	    // Go type: time
	    generated_at: any;

	    static createFrom(source: any = {}) {
	        return new BacktestReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.date = source["date"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.timeframe = source["timeframe"];
	        this.share_quantity = source["share_quantity"];
	        this.slippage_per_share = source["slippage_per_share"];
	        this.commission_per_order = source["commission_per_order"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.end_time = this.convertValues(source["end_time"], null);
	        this.bar_count = source["bar_count"];
	        this.trade_count = source["trade_count"];
	        this.winning_trades = source["winning_trades"];
	        this.losing_trades = source["losing_trades"];
	        this.win_rate_pct = source["win_rate_pct"];
	        this.total_gross_pnl = source["total_gross_pnl"];
	        this.total_commission = source["total_commission"];
	        this.total_net_pnl = source["total_net_pnl"];
	        this.total_return_pct = source["total_return_pct"];
	        this.max_drawdown = source["max_drawdown"];
	        this.trades = this.convertValues(source["trades"], BacktestTrade);
	        this.skipped_setups = this.convertValues(source["skipped_setups"], BacktestSkippedSetup);
	        this.skip_reason_counts = source["skip_reason_counts"];
	        this.open_position = this.convertValues(source["open_position"], BacktestOpenPosition);
	        this.generated_at = this.convertValues(source["generated_at"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BacktestRequest {
	    symbol: string;
	    date?: string;
	    start_date?: string;
	    end_date?: string;
	    start_time?: string;
	    end_time?: string;
	    timeframe?: string;
	    share_quantity: number;
	    slippage_per_share: number;
	    commission_per_order: number;

	    static createFrom(source: any = {}) {
	        return new BacktestRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbol = source["symbol"];
	        this.date = source["date"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.start_time = source["start_time"];
	        this.end_time = source["end_time"];
	        this.timeframe = source["timeframe"];
	        this.share_quantity = source["share_quantity"];
	        this.slippage_per_share = source["slippage_per_share"];
	        this.commission_per_order = source["commission_per_order"];
	    }
	}










}

