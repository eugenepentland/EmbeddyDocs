export namespace main {
	
	export class Embedding {
	    embedding: number[];
	    name: string;
	    length: number;
	
	    static createFrom(source: any = {}) {
	        return new Embedding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.embedding = source["embedding"];
	        this.name = source["name"];
	        this.length = source["length"];
	    }
	}
	export class Test {
	    id: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Test(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}

}

