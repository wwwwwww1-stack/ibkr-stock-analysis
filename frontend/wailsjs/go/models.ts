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
	    entry_zone?: EntryZone;
	    stop_loss?: number;
	    take_profit: number[];
	    risk_reward?: number;
	    confidence: number;
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
	        this.entry_zone = this.convertValues(source["entry_zone"], EntryZone);
	        this.stop_loss = source["stop_loss"];
	        this.take_profit = source["take_profit"];
	        this.risk_reward = source["risk_reward"];
	        this.confidence = source["confidence"];
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
	export class AnalysisResult {
	    symbol: string;
	    timeframe: string;
	    current_price: number;
	    output: AgentOutput;
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
	export class Settings {
	    ibkr_host: string;
	    ibkr_port: number;
	    ibkr_client_id: number;
	    watchlist: string[];
	    selected_timeframe: string;
	
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
	    }
	}
	export class AppState {
	    settings: Settings;
	    connection_status: string;
	    symbols: SymbolState[];
	    last_error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], Settings);
	        this.connection_status = source["connection_status"];
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
	
	

}

