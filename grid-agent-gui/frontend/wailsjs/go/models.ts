export namespace main {
	
	export class Step {
	    type: string;
	    commandID: string;
	    content: string;
	    output: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new Step(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.commandID = source["commandID"];
	        this.content = source["content"];
	        this.output = source["output"];
	        this.error = source["error"];
	    }
	}
	export class Message {
	    role: string;
	    content: string;
	    timestamp: string;
	    requestID: string;
	    steps: Step[];
	    isCommand: boolean;
	    output: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	        this.requestID = source["requestID"];
	        this.steps = this.convertValues(source["steps"], Step);
	        this.isCommand = source["isCommand"];
	        this.output = source["output"];
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
	    mnemonics: string;
	    network: string;
	    geminiApiKey: string;
	    theme: string;
	    isConfigured: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mnemonics = source["mnemonics"];
	        this.network = source["network"];
	        this.geminiApiKey = source["geminiApiKey"];
	        this.theme = source["theme"];
	        this.isConfigured = source["isConfigured"];
	    }
	}

}

