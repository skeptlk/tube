
export class Category {
    id: number;
    title: number;

    constructor(base: any = {}) {
        this.id = base['id'] || -1;
        this.title = base['title'] || '';
    }
}
