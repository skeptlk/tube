import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import { User, Video } from '../models';

@Injectable({ providedIn: 'root' })
export class AdminService {

    public BASE_URL = "http://localhost:8000/admin";

    constructor(private http: HttpClient) {}

    public getUsers (offset: number = 0, limit: number = 10) {
        return this.http
            .get<any>(`${this.BASE_URL}/user/?offset=${offset}&limit=${limit}`)
            .pipe(map(resp => {
                resp.users = resp.users.map(usr => new User(usr));
                return resp;
            }));
    }

    public deleteUser(id: number) {
        return this.http
            .delete<any>(`${this.BASE_URL}/user/${id}`)
            .pipe();
    }

    public getVideos(offset: number = 0, limit: number = 10) {
        return this.http
            .get<any>(`${this.BASE_URL}/video/?offset=${offset}&limit=${limit}`)
            .pipe(map(resp => {
                resp.videos = resp.users.map(vid => new Video(vid));
                return resp;
            }));
    }

    public deleteVideo() {

    }

}