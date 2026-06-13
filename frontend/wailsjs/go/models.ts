export namespace main {
	
	export class SnippetConfig {
	    label: string;
	    cmd: string;
	
	    static createFrom(source: any = {}) {
	        return new SnippetConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.cmd = source["cmd"];
	    }
	}
	export class Settings {
	    theme: string;
	    fontSize: number;
	    defaultShell: string;
	    sidebarOpenDefault: boolean;
	    snippets: SnippetConfig[];
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.fontSize = source["fontSize"];
	        this.defaultShell = source["defaultShell"];
	        this.sidebarOpenDefault = source["sidebarOpenDefault"];
	        this.snippets = this.convertValues(source["snippets"], SnippetConfig);
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

