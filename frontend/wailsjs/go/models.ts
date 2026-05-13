export namespace main {
	
	export class ClipboardResult {
	    input: string;
	    output: string;
	    inputFormat: string;
	
	    static createFrom(source: any = {}) {
	        return new ClipboardResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.output = source["output"];
	        this.inputFormat = source["inputFormat"];
	    }
	}

}

