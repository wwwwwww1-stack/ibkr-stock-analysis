export namespace domain {

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

