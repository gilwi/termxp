export namespace main {
	
	export class HistoryEntry {
	    id: string;
	    command: string;
	    host: string;
	    timestamp: number;
	
	    static createFrom(source: any = {}) {
	        return new HistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.command = source["command"];
	        this.host = source["host"];
	        this.timestamp = source["timestamp"];
	    }
	}

}

