export namespace main {
	
	export class SSHProfile {
	    name: string;
	    host: string;
	    user: string;
	    port: number;
	    keyPath: string;
	    jumpHost: string;
	
	    static createFrom(source: any = {}) {
	        return new SSHProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.host = source["host"];
	        this.user = source["user"];
	        this.port = source["port"];
	        this.keyPath = source["keyPath"];
	        this.jumpHost = source["jumpHost"];
	    }
	}
	export class WorkspaceNode {
	    type: string;
	    orientation?: string;
	    children?: WorkspaceNode[];
	    sizes?: number[];
	    shell?: string;
	    cwd?: string;
	    ssh?: SSHProfile;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.orientation = source["orientation"];
	        this.children = this.convertValues(source["children"], WorkspaceNode);
	        this.sizes = source["sizes"];
	        this.shell = source["shell"];
	        this.cwd = source["cwd"];
	        this.ssh = this.convertValues(source["ssh"], SSHProfile);
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
	export class WorkspaceTab {
	    name: string;
	    root: WorkspaceNode;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceTab(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.root = this.convertValues(source["root"], WorkspaceNode);
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
	export class Config {
	    version: number;
	    theme: string;
	    fontSize: number;
	    defaultShell: string;
	    copyOnSelect: boolean;
	    restoreWorkspace: boolean;
	    workspace: WorkspaceTab[];
	    sshProfiles: SSHProfile[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.theme = source["theme"];
	        this.fontSize = source["fontSize"];
	        this.defaultShell = source["defaultShell"];
	        this.copyOnSelect = source["copyOnSelect"];
	        this.restoreWorkspace = source["restoreWorkspace"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceTab);
	        this.sshProfiles = this.convertValues(source["sshProfiles"], SSHProfile);
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
	export class ConfigResult {
	    config: Config;
	    warning: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], Config);
	        this.warning = source["warning"];
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
	
	export class SessionOptions {
	    shell: string;
	    cwd: string;
	    ssh?: SSHProfile;
	
	    static createFrom(source: any = {}) {
	        return new SessionOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.shell = source["shell"];
	        this.cwd = source["cwd"];
	        this.ssh = this.convertValues(source["ssh"], SSHProfile);
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
	export class SystemStats {
	    cpu?: number;
	    memory?: number;
	    memoryRaw: string;
	    uptime: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpu = source["cpu"];
	        this.memory = source["memory"];
	        this.memoryRaw = source["memoryRaw"];
	        this.uptime = source["uptime"];
	    }
	}
	

}

