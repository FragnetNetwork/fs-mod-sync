export namespace config {
	
	export class Config {
	    serverUrl: string;
	    modsDirectory: string;
	    gameVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverUrl = source["serverUrl"];
	        this.modsDirectory = source["modsDirectory"];
	        this.gameVersion = source["gameVersion"];
	    }
	}

}

export namespace models {
	
	export class Mod {
	    name: string;
	    version: string;
	    author: string;
	    filename: string;
	    size: string;
	    sizeBytes: number;
	    isDLC: boolean;
	    isActive: boolean;
	    url: string;
	    needsUpdate: boolean;
	    localVersion?: string;
	
	    static createFrom(source: any = {}) {
	        return new Mod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.filename = source["filename"];
	        this.size = source["size"];
	        this.sizeBytes = source["sizeBytes"];
	        this.isDLC = source["isDLC"];
	        this.isActive = source["isActive"];
	        this.url = source["url"];
	        this.needsUpdate = source["needsUpdate"];
	        this.localVersion = source["localVersion"];
	    }
	}
	export class SyncStatus {
	    totalMods: number;
	    modsToSync: number;
	    totalSize: string;
	    totalSizeBytes: number;
	    gameVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalMods = source["totalMods"];
	        this.modsToSync = source["modsToSync"];
	        this.totalSize = source["totalSize"];
	        this.totalSizeBytes = source["totalSizeBytes"];
	        this.gameVersion = source["gameVersion"];
	    }
	}
	export class SyncResult {
	    status: SyncStatus;
	    mods: Mod[];
	
	    static createFrom(source: any = {}) {
	        return new SyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = this.convertValues(source["status"], SyncStatus);
	        this.mods = this.convertValues(source["mods"], Mod);
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
	
	export class ValidationResult {
	    valid: boolean;
	    gameVersion: string;
	    modCount: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.gameVersion = source["gameVersion"];
	        this.modCount = source["modCount"];
	        this.error = source["error"];
	    }
	}

}

