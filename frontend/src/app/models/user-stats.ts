
export class UserStats {
    id: number;
    name: string;
    password: string;
    email: string;
    token?: string;
    isAdmin: string;
    createdAt: Date;
    totalViews: number;
    numVideos: number;

    constructor(base: any) {
        this.id = base['id'];
        this.name = base['name'];
        this.password = base['password'];
        this.email = base['email'];
        this.isAdmin = base['isAdmin'];
        this.createdAt = base['createdAt'];
        this.totalViews = base['totalViews'];
        this.numVideos = base['numVideos'];
    }
}
