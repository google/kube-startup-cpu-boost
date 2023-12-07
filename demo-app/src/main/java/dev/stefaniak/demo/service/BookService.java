// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package com.example.demo.service;

import com.example.demo.db.BookRepository;
import java.util.List;
import java.util.Optional;
import java.util.stream.Collectors;
import org.springframework.stereotype.Service;

@Service
public class BookService {
  private BookRepository repository;

  public BookService(BookRepository repository) {
    this.repository = repository;
  }

  public List<Book> getAll() {
    return repository.findAll().stream()
        .map(b -> new Book(b.getId(), b.getTitle(), b.getAuthor(), b.getCategory()))
        .collect(Collectors.toList());
  }

  public Optional<Book> get(Long id) {
    return repository.findById(id)
        .map(b -> new Book(b.getId(), b.getTitle(), b.getAuthor(), b.getCategory()));
  }
}
